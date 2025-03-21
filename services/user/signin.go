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
	// 1. Fetch user record.
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to fetch user", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	if userRec == nil {
		utils.GetLogger().Debug("AuthenticateUser: No user found for email", zap.String("email", email))
		return nil, fmt.Errorf("invalid email or password")
	}
	utils.GetLogger().Debug("AuthenticateUser: Retrieved user", zap.String("userID", userRec.ID))

	// 2. Verify password.
	err = bcrypt.CompareHashAndPassword([]byte(userRec.PasswordHash), []byte(password))
	if err != nil {
		utils.GetLogger().Error("AuthenticateUser: Password verification failed", zap.String("userID", userRec.ID))
		return nil, fmt.Errorf("invalid email or password")
	}
	utils.GetLogger().Debug("AuthenticateUser: Password verified", zap.String("userID", userRec.ID))

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// 3. Determine session ID.
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
			utils.GetLogger().Error("AuthenticateUser: Failed to create auth session", zap.Error(err))
			return nil, fmt.Errorf("failed to create auth session: %w", err)
		}
		utils.GetLogger().Debug("AuthenticateUser: Created new auth session", zap.String("sessionID", sessionID))
	}

	// 4. Fetch the current auth session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to retrieve auth session", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve auth session: %w", err)
	}
	utils.GetLogger().Debug("AuthenticateUser: Auth session status", zap.String("status", authSession.Status))

	// 5. Check if the device is already registered.
	deviceExists := false
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			deviceExists = true
			// Update device details.
			userRec.Devices[idx].IP = currentDevice.IP
			userRec.Devices[idx].Location = currentDevice.Location
			break
		}
	}
	utils.GetLogger().Debug("AuthenticateUser: Device exists", zap.Bool("deviceExists", deviceExists))

	// 6. If device is not registered, handle OTP and append device.
	if !deviceExists {
		if authSession.Status != "otp_verified" {
			if len(userRec.Devices) >= 3 {
				utils.GetLogger().Error("AuthenticateUser: Maximum device limit reached", zap.Int("deviceCount", len(userRec.Devices)))
				return nil, fmt.Errorf("maximum device limit reached. Only 3 devices are allowed")
			}
			otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
			_, err := sessionClient.Get(ctx, otpCacheKey).Result()
			if err != nil {
				if err := utils.InitiateDeviceOTP(userRec.ID, currentDevice.DeviceID, userRec.PhoneNumber); err != nil {
					utils.GetLogger().Error("AuthenticateUser: Failed to initiate OTP", zap.Error(err))
					return nil, fmt.Errorf("failed to initiate OTP: %w", err)
				}
				authSession.Status = "pending_otp"
				if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
					utils.GetLogger().Error("AuthenticateUser: Failed to update auth session after OTP initiation", zap.Error(err))
					return nil, fmt.Errorf("failed to update auth session: %w", err)
				}
				utils.GetLogger().Debug("AuthenticateUser: OTP initiated for new device", zap.String("sessionID", sessionID))
			}
			return nil, OTPPendingError{SessionID: sessionID}
		}
		// OTP verified: append the new device.
		currentDevice.LastLogin = time.Now()
		currentDevice.Creator = false
		userRec.Devices = append(userRec.Devices, currentDevice)
		utils.GetLogger().Debug("AuthenticateUser: New device appended", zap.String("deviceID", currentDevice.DeviceID))
	}

	// 7. Generate a new JWT token for this device.
	token, err := utils.GenerateToken(userRec.ID, userRec.Email, currentDevice.DeviceID)
	if err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to generate token", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	tokenHash := utils.HashToken(token)
	utils.GetLogger().Debug("AuthenticateUser: Token generated and hashed", zap.String("userID", userRec.ID))

	// 8. Update the token hash and LastLogin for the matching device.
	for idx, d := range userRec.Devices {
		if d.DeviceID == currentDevice.DeviceID {
			userRec.Devices[idx].TokenHash = tokenHash
			userRec.Devices[idx].LastLogin = time.Now()
			break
		}
	}
	utils.GetLogger().Debug("AuthenticateUser: Updated device info", zap.String("deviceID", currentDevice.DeviceID))

	// 9. Update the user record.
	updateDoc := bson.M{
		"$set": bson.M{
			"devices":    userRec.Devices,
			"updated_at": time.Now(),
		},
	}
	if err := s.Repo.UpdateWithDocument(userRec.ID, updateDoc); err != nil {
		utils.GetLogger().Error("AuthenticateUser: Failed to update user record", zap.Error(err))
		return nil, fmt.Errorf("authentication failed, please try again")
	}
	utils.GetLogger().Debug("AuthenticateUser: User record updated", zap.String("userID", userRec.ID))

	// 10. Clear the auth session.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)
	utils.GetLogger().Debug("AuthenticateUser: Cleared auth session", zap.String("sessionID", sessionID))

	// 11. Return the auth response.
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
