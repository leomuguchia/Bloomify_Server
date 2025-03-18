// File: bloomify/service/user/user.go
package user

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthenticateUser authenticates the user with device info using an auth session.
func (s *DefaultUserService) AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error) {
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("Failed to fetch user", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	if userRec == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Always verify the password.
	if err := bcrypt.CompareHashAndPassword([]byte(userRec.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// Determine session ID. If providedSessionID is not empty, use it; otherwise, create one.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", userRec.ID, currentDevice.DeviceID)
		// Create a new auth session with status "pending"
		authSession := utils.AuthSession{
			UserID:        userRec.ID,
			Email:         userRec.Email,
			Device:        utils.DeviceSessionInfo{DeviceID: currentDevice.DeviceID, DeviceName: currentDevice.DeviceName, IP: currentDevice.IP, Location: currentDevice.Location},
			Status:        "pending",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Username:      userRec.Username,
			PhoneNumber:   userRec.PhoneNumber,
			Rating:        userRec.Rating,
		}
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
	}

	// Fetch the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}

	// Check if the device is already registered.
	deviceExists := false
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			// Update device details.
			userRec.Devices[idx].IP = currentDevice.IP
			userRec.Devices[idx].Location = currentDevice.Location
			userRec.Devices[idx].LastLogin = time.Now()
			break
		}
	}

	// If the device is not registered, decide whether to initiate OTP or continue.
	if !deviceExists {
		// If the session status is not "otp_verified", then initiate OTP if needed.
		if authSession.Status != "otp_verified" {
			// Enforce maximum device limit.
			if len(userRec.Devices) >= 3 {
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			// Check if an OTP is already in cache.
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				// OTP not set; initiate OTP.
				if err := utils.InitiateDeviceOTP(userRec.ID, currentDevice.DeviceID, userRec.PhoneNumber); err != nil {
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

		// If OTP has been verified (status is "otp_verified"), add the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		userRec.Devices = append(userRec.Devices, currentDevice)
		if err := s.Repo.Update(userRec); err != nil {
			return nil, fmt.Errorf("failed to add new device: %w", err)
		}
	}

	// At this point, either the device was already registered,
	// or a new device has been added after OTP verification.
	// Proceed to generate a token and complete authentication.
	token, err := utils.GenerateToken(userRec.ID, userRec.Email, 24*time.Hour)
	if err != nil {
		utils.GetLogger().Error("Failed to generate token", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	userRec.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(userRec); err != nil {
		utils.GetLogger().Error("Failed to update user with token hash", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Clear the auth session since authentication is complete.
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
