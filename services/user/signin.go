package user

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

func (s *DefaultUserService) AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error) {
	// Fetch user record.
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to fetch user", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	if userRec == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password.
	err = bcrypt.CompareHashAndPassword([]byte(userRec.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// Determine session ID.
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
			Username:      userRec.Username,
			PhoneNumber:   userRec.PhoneNumber,
			Rating:        userRec.Rating,
		}
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
	}

	// Fetch the current auth session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}

	// Check if the device is already registered.
	deviceExists := false
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			userRec.Devices[idx].IP = currentDevice.IP
			userRec.Devices[idx].Location = currentDevice.Location
			break
		}
	}

	// If device is not registered, handle OTP and append device.
	if !deviceExists {
		if authSession.Status != "otp_verified" {
			if len(userRec.Devices) >= 3 {
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				if err := utils.InitiateDeviceOTP(userRec.ID, currentDevice.DeviceID, userRec.PhoneNumber); err != nil {
					return nil, fmt.Errorf("failed to initiate OTP: %w", err)
				}
				authSession.Status = "pending_otp"
				if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
					return nil, fmt.Errorf("failed to update auth session: %w", err)
				}
			}
			return nil, OTPPendingError{SessionID: sessionID}
		}
		// OTP verified: append the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		userRec.Devices = append(userRec.Devices, currentDevice)
	}

	// Clear any stale token hash for this device.
	cacheKey := utils.AuthCachePrefix + userRec.ID + ":" + currentDevice.DeviceID
	if err := sessionClient.Del(ctx, cacheKey).Err(); err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to clear old token cache", zap.Error(err))
	}

	// Generate a new JWT token for this device.
	token, err := utils.GenerateToken(userRec.ID, userRec.Email, currentDevice.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	tokenHash := utils.HashToken(token)

	// Update the token hash and LastLogin for the matching device.
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			userRec.Devices[idx].TokenHash = tokenHash
			userRec.Devices[idx].LastLogin = time.Now()
			break
		}
	}

	// Update the user record.
	updateDoc := bson.M{
		"$set": bson.M{
			"devices":    userRec.Devices,
			"updated_at": time.Now(),
		},
	}
	if err := s.Repo.UpdateWithDocument(userRec.ID, updateDoc); err != nil {
		return nil, fmt.Errorf("authentication failed, please try again")
	}

	// Clear the auth session.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	// Return the auth response.
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
