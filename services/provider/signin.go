package provider

import (
	"bloomify/models"
	"bloomify/services/socialAuth.go"
	"bloomify/utils"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// InitiateProviderAuthentication handles the first step of provider authentication
func (s *DefaultProviderService) InitiateProviderAuthentication(email, method, password string, currentDevice models.Device) (*models.ProviderAuthResponse, string, int, error) {
	email = strings.ToLower(email)
	// Fetch provider details
	projection := bson.M{
		"security.passwordHash": 1,
		"id":                    1,
		"profile.email":         1,
		"profile.phoneNumber":   1,
		"devices":               1,
	}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		utils.GetLogger().Error("Failed to fetch provider", zap.Error(err))
		return nil, "", 0, fmt.Errorf("authentication failed, please try again")
	}
	if provider == nil {
		return nil, "", 0, fmt.Errorf("invalid email or password")
	}

	// Handle different authentication methods
	switch method {
	case "password":
		if err := bcrypt.CompareHashAndPassword([]byte(provider.Security.PasswordHash), []byte(password)); err != nil {
			utils.GetLogger().Error("Password hash mismatch", zap.String("hash", provider.Security.PasswordHash), zap.String("provided", password), zap.Error(err))
			return nil, "", 0, fmt.Errorf("invalid email or password")
		}
	case "apple":
		// Validate Apple token (password contains the Apple ID token)
		if password == "" {
			return nil, "", 0, fmt.Errorf("apple token is required")
		}
		userInfo, err := socialAuth.ValidateAppleToken(password, "com.your.app.bundleid")
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid apple token: %v", err)
		}
		if userInfo.Email != email {
			return nil, "", 0, fmt.Errorf("email doesn't match apple account")
		}
	case "google":
		// Validate Google token (password contains the Google ID token)
		if password == "" {
			return nil, "", 0, fmt.Errorf("google token is required")
		}
		userInfo, err := socialAuth.ValidateGoogleToken(password, "your-google-client-id")
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid google token: %v", err)
		}
		if userInfo.Email != email {
			return nil, "", 0, fmt.Errorf("email doesn't match google account")
		}
	case "facebook":
		// Validate Facebook token (password contains the Facebook access token)
		if password == "" {
			return nil, "", 0, fmt.Errorf("facebook token is required")
		}
		userInfo, err := socialAuth.ValidateFacebookToken(password, "your-facebook-app-id", "your-facebook-app-secret")
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid facebook token: %v", err)
		}
		if userInfo.Email != email {
			return nil, "", 0, fmt.Errorf("email doesn't match facebook account")
		}
	default:
		return nil, "", 0, fmt.Errorf("unsupported authentication method")
	}

	sessionClient := utils.GetProviderAuthCacheClient()
	sessionID := fmt.Sprintf("%s:%s", provider.ID, currentDevice.DeviceID)

	// Check if device is already registered
	deviceExists := false
	for _, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			break
		}
	}

	if deviceExists {
		// Device is known - proceed with immediate authentication
		authResp, err := s.completeProviderAuthentication(provider.ID, currentDevice)
		if err != nil {
			return nil, "", 0, err
		}
		return authResp, "", 0, nil
	}

	// New device requires OTP verification
	authSession := utils.AuthSession{
		UserID: provider.ID,
		Email:  provider.Profile.Email,
		Device: utils.DeviceSessionInfo{
			DeviceID:   currentDevice.DeviceID,
			DeviceName: currentDevice.DeviceName,
			IP:         currentDevice.IP,
			Location:   currentDevice.Location,
		},
		Status:        "pending_otp",
		CreatedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
	}

	if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
		return nil, "", 0, fmt.Errorf("failed to create auth session: %w", err)
	}

	if err := utils.InitiateDeviceOTP(provider.ID, currentDevice.DeviceID, provider.Profile.PhoneNumber); err != nil {
		return nil, "", 0, fmt.Errorf("failed to initiate OTP: %w", err)
	}

	return nil, sessionID, 100, nil
}

// CheckProviderAuthenticationStatus returns the current status of an authentication session
func (s *DefaultProviderService) CheckProviderAuthenticationStatus(sessionID string) (string, error) {
	sessionClient := utils.GetProviderAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid or expired session")
	}
	return authSession.Status, nil
}

// VerifyProviderAuthenticationOTP verifies the OTP and completes authentication
func (s *DefaultProviderService) VerifyProviderAuthenticationOTP(sessionID, otp string, currentDevice models.Device) (*models.ProviderAuthResponse, error) {
	sessionClient := utils.GetProviderAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Verify OTP
	if err := utils.VerifyDeviceOTPRecord(authSession.UserID, currentDevice.DeviceID, otp); err != nil {
		return nil, fmt.Errorf("OTP verification failed: %w", err)
	}

	// Fetch full provider record
	provider, err := s.Repo.GetByIDWithProjection(authSession.UserID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Update session status
	authSession.Status = "otp_verified"
	if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
		return nil, fmt.Errorf("failed to update auth session: %w", err)
	}

	// Complete authentication
	return s.completeProviderAuthentication(provider.ID, currentDevice)
}

func (s *DefaultProviderService) completeProviderAuthentication(providerId string, currentDevice models.Device) (*models.ProviderAuthResponse, error) {
	sessionClient := utils.GetProviderAuthCacheClient()
	sessionID := fmt.Sprintf("%s:%s", providerId, currentDevice.DeviceID)

	// Fetch full provider record
	provider, err := s.Repo.GetByIDWithProjection(providerId, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Check if device is already registered
	deviceExists := false
	for idx, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			provider.Devices[idx].IP = currentDevice.IP
			provider.Devices[idx].Location = currentDevice.Location
			break
		}
	}

	// If device is not registered, add it
	if !deviceExists {
		if len(provider.Devices) >= 3 {
			return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
		}
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		provider.Devices = append(provider.Devices, currentDevice)
	}

	// Generate new token
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, currentDevice.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	tokenHash := utils.HashToken(token)

	// clear any existing token for the device
	if sessionClient != nil {
		cacheKey := utils.ProviderAuthCachePrefix + providerId + ":" + currentDevice.DeviceID
		if err := sessionClient.Del(context.Background(), cacheKey).Err(); err != nil {
			log.Printf("Failed to clear stale token cache: %v", err)
		}
	}

	// Update device token hash & last login time
	for idx, d := range provider.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			provider.Devices[idx].TokenHash = tokenHash
			provider.Devices[idx].LastLogin = time.Now()
			break
		}
	}

	// Update provider record in DB
	updateDoc := bson.M{
		"devices":   provider.Devices,
		"updatedAt": time.Now(),
	}
	if err := s.Repo.UpdateSetDocument(provider.ID, updateDoc); err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Clear the auth session (e.g. OTP/session tracking)
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	return &models.ProviderAuthResponse{
		ID:               provider.ID,
		Token:            token,
		Profile:          provider.Profile,
		CreatedAt:        provider.CreatedAt,
		ServiceCatalogue: provider.ServiceCatalogue,
	}, nil
}
