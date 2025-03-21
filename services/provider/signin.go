package provider

import (
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthenticateProvider verifies credentials, handles OTP for new devices,
// updates the per‑device token hash, and returns an enriched auth response.
func (s *DefaultProviderService) AuthenticateProvider(email, password string, currentDevice models.Device, providedSessionID string) (*models.ProviderAuthResponse, error) {
	// 1. Fetch provider details using a projection.
	projection := bson.M{
		"password_hash": 1,
		"id":            1,
		"email":         1,
		"devices":       1,
		"profile":       1,
	}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		utils.GetLogger().Error("Failed to fetch provider", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	if provider == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// 2. Verify the password.
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// 3. Get the OTP/Session Redis client and context.
	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// 4. Determine the session ID.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", provider.ID, currentDevice.DeviceID)
		authSession := utils.AuthSession{
			UserID:        provider.ID,
			Email:         provider.Profile.Email,
			Device:        utils.DeviceSessionInfo{DeviceID: currentDevice.DeviceID, DeviceName: currentDevice.DeviceName, IP: currentDevice.IP, Location: currentDevice.Location},
			Status:        "pending",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
		}
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
	}

	// 5. Retrieve the current auth session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}

	// 6. Check if the device is already registered.
	deviceExists := false
	for idx, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			// Update device details.
			provider.Devices[idx].IP = currentDevice.IP
			provider.Devices[idx].Location = currentDevice.Location
			// We'll update LastLogin along with token hash.
			break
		}
	}

	// 7. If the device is not registered.
	if !deviceExists {
		// If OTP is not verified, enforce OTP flow.
		if authSession.Status != "otp_verified" {
			if len(provider.Devices) >= 3 {
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				// OTP not set; initiate OTP.
				if err := utils.InitiateDeviceOTP(provider.ID, currentDevice.DeviceID, provider.Profile.PhoneNumber); err != nil {
					return nil, fmt.Errorf("failed to initiate OTP: %w", err)
				}
				authSession.Status = "pending_otp"
				if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
					return nil, fmt.Errorf("failed to update auth session: %w", err)
				}
			}
			return nil, OTPPendingError{SessionID: sessionID}
		}
		// OTP is verified; append the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		provider.Devices = append(provider.Devices, currentDevice)
	}

	// 8. Generate a new JWT token for this device (including the device ID).
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, currentDevice.DeviceID)
	if err != nil {
		utils.GetLogger().Error("Failed to generate token", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	tokenHash := utils.HashToken(token)

	// 9. Update the token hash and LastLogin for the matching device.
	deviceUpdated := false
	for idx, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			provider.Devices[idx].TokenHash = tokenHash
			provider.Devices[idx].LastLogin = time.Now()
			deviceUpdated = true
			break
		}
	}
	// Defensive fallback: if not updated, append the device.
	if !deviceUpdated {
		currentDevice.TokenHash = tokenHash
		currentDevice.LastLogin = time.Now()
		provider.Devices = append(provider.Devices, currentDevice)
	}

	// 10. Build an update document to patch the devices field and updated timestamp in one call.
	updateDoc := bson.M{
		"$set": bson.M{
			"devices":    provider.Devices,
			"updated_at": time.Now(),
		},
	}
	if err := s.Repo.UpdateWithDocument(provider.ID, updateDoc); err != nil {
		utils.GetLogger().Error("Failed to update provider with device token hash", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// 11. Clear the auth session since authentication is complete.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	// 12. Build and return the enriched authentication response.
	return &models.ProviderAuthResponse{
		ID:           provider.ID,
		Token:        token,
		Profile:      provider.Profile,
		CreatedAt:    provider.CreatedAt,
		ProviderType: provider.ProviderType,
		ServiceType:  provider.ServiceCatalogue.ServiceType,
		Rating:       provider.Rating,
	}, nil
}
