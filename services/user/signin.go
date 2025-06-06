package user

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

// InitiateAuthentication handles the first step of authentication
func (s *DefaultUserService) InitiateAuthentication(email, method, password string, currentDevice models.Device) (*AuthResponse, string, int, error) {
	emailLower := strings.ToLower(email)
	// Fetch user record
	userRec, err := s.Repo.GetByEmailWithProjection(emailLower, bson.M{})
	if err != nil {
		utils.GetLogger().Error("InitiateAuthentication: Failed to fetch user", zap.Error(err))
		return nil, "", 0, fmt.Errorf("authentication failed, please try again")
	}
	if userRec == nil {
		return nil, "", 0, fmt.Errorf("invalid email or password")
	}

	// Handle different authentication methods
	switch method {
	case "password":
		if err := bcrypt.CompareHashAndPassword([]byte(userRec.PasswordHash), []byte(password)); err != nil {
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
		if strings.ToLower(userInfo.Email) != emailLower {
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
		if strings.ToLower(userInfo.Email) != emailLower {
			return nil, "", 0, fmt.Errorf("email doesn't match google account")
		}
	default:
		return nil, "", 0, fmt.Errorf("unsupported authentication method")
	}

	sessionClient := utils.GetAuthCacheClient()
	sessionID := fmt.Sprintf("%s:%s", userRec.ID, currentDevice.DeviceID)

	// Check if device is already registered
	deviceExists := false
	for _, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			break
		}
	}

	if deviceExists {
		// Device is known - proceed with immediate authentication
		authResp, err := s.completeAuthentication(userRec, currentDevice)
		if err != nil {
			return nil, "", 0, err
		}
		return authResp, "", 0, nil
	}

	// New device requires OTP verification
	authSession := utils.AuthSession{
		UserID:        userRec.ID,
		Email:         userRec.Email,
		Device:        utils.DeviceSessionInfo{DeviceID: currentDevice.DeviceID, DeviceName: currentDevice.DeviceName, IP: currentDevice.IP, Location: currentDevice.Location},
		Status:        "pending_otp",
		CreatedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
		Username:      userRec.Username,
		PhoneNumber:   userRec.PhoneNumber,
		Rating:        userRec.Rating,
	}

	if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
		return nil, "", 0, fmt.Errorf("failed to create auth session: %w", err)
	}

	if err := utils.InitiateDeviceOTP(userRec.ID, currentDevice.DeviceID, userRec.PhoneNumber); err != nil {
		return nil, "", 0, fmt.Errorf("failed to initiate OTP: %w", err)
	}

	return nil, sessionID, 100, nil
}

// CheckAuthenticationStatus returns the current status of an authentication session
func (s *DefaultUserService) CheckAuthenticationStatus(sessionID string) (string, error) {
	sessionClient := utils.GetAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid or expired session")
	}
	return authSession.Status, nil
}

// VerifyAuthenticationOTP verifies the OTP and completes authentication
func (s *DefaultUserService) VerifyAuthenticationOTP(sessionID, otp string, currentDevice models.Device) (*AuthResponse, error) {
	sessionClient := utils.GetAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Verify OTP
	if err := utils.VerifyDeviceOTPRecord(authSession.UserID, currentDevice.DeviceID, otp); err != nil {
		return nil, fmt.Errorf("OTP verification failed: %w", err)
	}

	// Fetch user record
	userRec, err := s.Repo.GetByIDWithProjection(authSession.UserID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Update session status
	authSession.Status = "otp_verified"
	if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
		return nil, fmt.Errorf("failed to update auth session: %w", err)
	}

	// Complete authentication
	return s.completeAuthentication(userRec, currentDevice)
}

// completeAuthentication handles the final steps of authentication
func (s *DefaultUserService) completeAuthentication(userRec *models.User, currentDevice models.Device) (*AuthResponse, error) {
	sessionClient := utils.GetAuthCacheClient()
	sessionID := fmt.Sprintf("%s:%s", userRec.ID, currentDevice.DeviceID)

	// Check if device is already registered
	deviceExists := false
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			userRec.Devices[idx].IP = currentDevice.IP
			userRec.Devices[idx].Location = currentDevice.Location
			break
		}
	}

	// If device is not registered, add it
	if !deviceExists {
		if len(userRec.Devices) >= 3 {
			return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
		}
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		userRec.Devices = append(userRec.Devices, currentDevice)
	}

	// Generate new token
	token, err := utils.GenerateToken(userRec.ID, userRec.Email, currentDevice.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	tokenHash := utils.HashToken(token)

	// clear any existing token for the device
	if sessionClient != nil {
		cacheKey := utils.AuthCachePrefix + userRec.ID + ":" + currentDevice.DeviceID
		if err := sessionClient.Del(context.Background(), cacheKey).Err(); err != nil {
			log.Printf("Failed to clear stale token cache: %v", err)
		}
	}

	// Update device token
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			userRec.Devices[idx].TokenHash = tokenHash
			userRec.Devices[idx].LastLogin = time.Now()
			break
		}
	}

	// Update user record
	updateDoc := bson.M{
		"devices":   userRec.Devices,
		"updatedAt": time.Now(),
	}
	if err := s.Repo.UpdateSetDocument(userRec.ID, updateDoc); err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Clear the auth session
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	return &AuthResponse{
		ID:           userRec.ID,
		Token:        token,
		Username:     userRec.Username,
		Email:        userRec.Email,
		PhoneNumber:  userRec.PhoneNumber,
		ProfileImage: userRec.ProfileImage,
		Rating:       userRec.Rating,
	}, nil
}
