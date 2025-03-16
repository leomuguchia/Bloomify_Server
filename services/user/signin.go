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

// OTPPendingError indicates that OTP verification is required.
type OTPPendingError struct {
	SessionID string
}

func (e OTPPendingError) Error() string {
	return fmt.Sprintf("OTP verification required. SessionID: %s", e.SessionID)
}

// AuthenticateUser authenticates the user with device info using an auth session.
func (s *DefaultUserService) AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error) {
	// Retrieve the full user record including sensitive fields.
	projection := bson.M{
		"password_hash": 1,
		"id":            1,
		"email":         1,
		"username":      1,
		"profile_image": 1,
		"phone_number":  1,
		"devices":       1,
	}
	userRec, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		utils.GetLogger().Error("Failed to fetch user", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	if userRec == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(userRec.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Get the OTP/Session Redis client.
	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// Determine the waiting session ID.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", userRec.ID, currentDevice.DeviceID)
		authSession := utils.AuthSession{
			UserID:        userRec.ID,
			Email:         userRec.Email,
			Device:        utils.DeviceSessionInfo{DeviceID: currentDevice.DeviceID, DeviceName: currentDevice.DeviceName, IP: currentDevice.IP, Location: currentDevice.Location},
			Status:        "pending",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
		}
		// Save the session.
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
	}

	// Retrieve the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}

	// If device not registered, check OTP verification status.
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

	if !deviceExists {
		// If OTP is not verified yet, check session status.
		if authSession.Status != "otp_verified" {
			// Before initiating OTP, enforce the maximum device limit.
			if len(userRec.Devices) >= 3 {
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			// Check if OTP is already cached.
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				// If OTP not set, initiate OTP.
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
		// If OTP is verified, add the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		userRec.Devices = append(userRec.Devices, currentDevice)
		if err := s.Repo.Update(userRec); err != nil {
			return nil, fmt.Errorf("failed to add new device: %w", err)
		}
	}

	// Now, complete authentication.
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

	// Clear session since authentication is complete.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	return &AuthResponse{
		ID:           userRec.ID,
		Token:        token,
		Username:     userRec.Username,
		Email:        userRec.Email,
		PhoneNumber:  userRec.PhoneNumber,
		ProfileImage: userRec.ProfileImage,
	}, nil
}
