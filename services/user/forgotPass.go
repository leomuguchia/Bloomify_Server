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

func (s *DefaultUserService) ResetPassword(email, providedOTP, newPassword, providedSessionID, currentDeviceID string) error {
	// Retrieve the user record by email.
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to fetch user", zap.Error(err))
		return fmt.Errorf("failed to reset password, please try again")
	}
	if userRec == nil {
		utils.GetLogger().Debug("ResetPassword: No user found for email", zap.String("email", email))
		return fmt.Errorf("invalid email")
	}
	utils.GetLogger().Debug("ResetPassword: Retrieved user", zap.String("userID", userRec.ID))

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// Determine session ID.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", userRec.ID, "reset_password")
		authSession := utils.AuthSession{
			UserID:        userRec.ID,
			Email:         userRec.Email,
			Status:        "pending",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
		}
		if err := utils.SaveAuthSession(sessionClient, sessionID, authSession); err != nil {
			utils.GetLogger().Error("ResetPassword: Failed to create auth session", zap.Error(err))
			return fmt.Errorf("failed to create password reset session: %w", err)
		}
		utils.GetLogger().Debug("ResetPassword: Created new auth session", zap.String("sessionID", sessionID))
	}

	// Fetch the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to retrieve auth session", zap.Error(err))
		return fmt.Errorf("failed to retrieve password reset session: %w", err)
	}
	utils.GetLogger().Debug("ResetPassword: Auth session status", zap.String("status", authSession.Status))

	// State 1: Initiate OTP if neither OTP nor new password is provided.
	if providedOTP == "" && newPassword == "" {
		otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
		_, err := sessionClient.Get(ctx, otpCacheKey).Result()
		if err != nil {
			if err := utils.InitiateDeviceOTP(userRec.ID, "reset_password", userRec.PhoneNumber); err != nil {
				utils.GetLogger().Error("ResetPassword: Failed to initiate OTP", zap.Error(err))
				return fmt.Errorf("failed to initiate OTP: %w", err)
			}
			authSession.Status = "pending_otp"
			if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
				utils.GetLogger().Error("ResetPassword: Failed to update auth session for OTP", zap.Error(err))
				return fmt.Errorf("failed to update password reset session: %w", err)
			}
			utils.GetLogger().Debug("ResetPassword: OTP initiated", zap.String("sessionID", sessionID))
		}
		return OTPPendingError{SessionID: sessionID}
	}

	// Verify OTP if not already verified.
	if authSession.Status != "otp_verified" {
		if err := utils.VerifyDeviceOTPRecord(userRec.ID, "reset_password", providedOTP); err != nil {
			utils.GetLogger().Error("ResetPassword: OTP verification failed", zap.Error(err))
			return fmt.Errorf("OTP verification failed: %w", err)
		}
		authSession.Status = "otp_verified"
		authSession.LastUpdatedAt = time.Now()
		if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
			utils.GetLogger().Error("ResetPassword: Failed to update auth session post OTP", zap.Error(err))
			return fmt.Errorf("failed to update password reset session: %w", err)
		}
		utils.GetLogger().Debug("ResetPassword: OTP verified", zap.String("sessionID", sessionID))
	}

	// State 2: If OTP is verified but no new password is provided, prompt for new password.
	if newPassword == "" {
		utils.GetLogger().Debug("ResetPassword: New password required", zap.String("sessionID", sessionID))
		return NewPasswordRequiredError{SessionID: sessionID}
	}

	// Validate and hash the new password.
	if err := VerifyPasswordComplexity(newPassword); err != nil {
		utils.GetLogger().Error("ResetPassword: New password failed complexity requirements", zap.Error(err))
		return err
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to hash new password", zap.Error(err))
		return fmt.Errorf("failed to process new password")
	}
	utils.GetLogger().Debug("ResetPassword: New password hashed successfully")

	// Build the update document.
	now := time.Now()
	updateFields := bson.M{
		"password_hash": string(newHash),
		"updated_at":    now,
	}

	// Retain only the current device if more than one device exists.
	if len(userRec.Devices) > 1 {
		var retainedDevices []models.Device
		authCache := utils.GetAuthCacheClient()
		for _, d := range userRec.Devices {
			if d.DeviceID == currentDeviceID {
				retainedDevices = append(retainedDevices, d)
			} else {
				cacheKey := utils.AuthCachePrefix + userRec.ID + ":" + d.DeviceID
				_ = authCache.Del(context.Background(), cacheKey).Err()
				utils.GetLogger().Debug("ResetPassword: Removed device from cache", zap.String("deviceID", d.DeviceID))
			}
		}
		updateFields["devices"] = retainedDevices
	} else {
		updateFields["devices"] = userRec.Devices
	}
	utils.GetLogger().Debug("ResetPassword: Prepared update fields", zap.Any("updateFields", updateFields))

	updateDoc := bson.M{"$set": updateFields}
	if err := s.Repo.UpdateWithDocument(userRec.ID, updateDoc); err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to update user password", zap.Error(err))
		return fmt.Errorf("failed to update password")
	}

	_ = utils.DeleteAuthSession(sessionClient, sessionID)
	utils.GetLogger().Sugar().Infof("ResetPassword: Password updated for user %s", userRec.ID)
	return nil
}
