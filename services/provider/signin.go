package provider

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// OTPPendingError indicates that OTP verification is required.
type OTPPendingError struct {
	SessionID string
}

func (e OTPPendingError) Error() string {
	return fmt.Sprintf("OTP verification required. SessionID: %s", e.SessionID)
}

// AuthenticateProvider verifies credentials, handles device OTP for new devices,
// updates token hash, and returns an enriched auth response.
func (s *DefaultProviderService) AuthenticateProvider(email, password string, currentDevice models.Device, providedSessionID string) (*ProviderAuthResponse, error) {
	// Fetch provider details (with required fields) using a projection.
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

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Get the OTP/Session Redis client.
	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// Determine the session ID.
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
		// Save the new auth session.
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
	}

	// Retrieve the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}

	// Check if the device is already registered.
	deviceExists := false
	for idx, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			// Update device details.
			provider.Devices[idx].IP = currentDevice.IP
			provider.Devices[idx].Location = currentDevice.Location
			provider.Devices[idx].LastLogin = time.Now()
			break
		}
	}

	// If the device is not registered.
	if !deviceExists {
		// If OTP is not verified yet, check session status.
		if authSession.Status != "otp_verified" {
			// Before initiating OTP, enforce the maximum device limit.
			if len(provider.Devices) >= 3 {
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			// Initiate OTP if not already initiated.
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				// If OTP is not set, initiate it.
				if err := utils.InitiateDeviceOTP(provider.ID, currentDevice.DeviceID, provider.Profile.PhoneNumber); err != nil {
					return nil, fmt.Errorf("failed to initiate OTP: %w", err)
				}
				authSession.Status = "pending_otp"
				if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
					return nil, fmt.Errorf("failed to update auth session: %w", err)
				}
			}
			// Return an OTP pending error with the sessionID.
			return nil, OTPPendingError{SessionID: sessionID}
		}
		// OTP has been verified. Check device limit again before adding.
		if len(provider.Devices) >= 3 {
			return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
		}
		// Add the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		provider.Devices = append(provider.Devices, currentDevice)
		if err := s.Repo.Update(provider); err != nil {
			return nil, fmt.Errorf("failed to add new device: %w", err)
		}
	}

	// Now, complete authentication by generating a new JWT token.
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, 24*time.Hour)
	if err != nil {
		utils.GetLogger().Error("Failed to generate token", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Update provider's token hash.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(provider); err != nil {
		utils.GetLogger().Error("Failed to update provider with token hash", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Clear the auth session since authentication is complete.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	// Build and return the enriched authentication response.
	return &ProviderAuthResponse{
		ID:           provider.ID,
		Token:        token,
		Profile:      provider.Profile,
		CreatedAt:    provider.CreatedAt,
		ProviderType: provider.ProviderType,
		ServiceType:  provider.ServiceType,
		Rating:       provider.Rating,
	}, nil
}
