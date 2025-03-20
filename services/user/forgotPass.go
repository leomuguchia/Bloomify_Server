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
		return fmt.Errorf("invalid email")
	}

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
			return fmt.Errorf("failed to create password reset session: %w", err)
		}
	}

	// Fetch the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve password reset session: %w", err)
	}

	// State 1: Initiate OTP if neither OTP nor new password is provided.
	if providedOTP == "" && newPassword == "" {
		otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
		_, err := sessionClient.Get(ctx, otpCacheKey).Result()
		if err != nil {
			if err := utils.InitiateDeviceOTP(userRec.ID, "reset_password", userRec.PhoneNumber); err != nil {
				return fmt.Errorf("failed to initiate OTP: %w", err)
			}
			authSession.Status = "pending_otp"
			if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
				return fmt.Errorf("failed to update password reset session: %w", err)
			}
		}
		return OTPPendingError{SessionID: sessionID}
	}

	// Verify OTP if not already verified.
	if authSession.Status != "otp_verified" {
		if err := utils.VerifyDeviceOTPRecord(userRec.ID, "reset_password", providedOTP); err != nil {
			return fmt.Errorf("OTP verification failed: %w", err)
		}
		authSession.Status = "otp_verified"
		authSession.LastUpdatedAt = time.Now()
		if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
			return fmt.Errorf("failed to update password reset session: %w", err)
		}
	}

	// State 2: If OTP is verified but no new password is provided, prompt for new password.
	if newPassword == "" {
		return NewPasswordRequiredError{SessionID: sessionID}
	}

	// Validate and hash the new password.
	if err := VerifyPasswordComplexity(newPassword); err != nil {
		return err
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to hash new password", zap.Error(err))
		return fmt.Errorf("failed to process new password")
	}

	// Build the update document: update password hash, updated_at, and filter devices.
	now := time.Now()
	updateFields := bson.M{
		"password_hash": string(newHash),
		"updated_at":    now,
	}

	// If there is more than one device, retain only the current device.
	if len(userRec.Devices) > 1 {
		var retainedDevices []models.Device
		authCache := utils.GetAuthCacheClient()
		for _, d := range userRec.Devices {
			if d.DeviceID == currentDeviceID {
				retainedDevices = append(retainedDevices, d)
			} else {
				cacheKey := utils.AuthCachePrefix + userRec.ID + ":" + d.DeviceID
				_ = authCache.Del(context.Background(), cacheKey).Err()
			}
		}
		updateFields["devices"] = retainedDevices
	} else {
		updateFields["devices"] = userRec.Devices
	}

	updateDoc := bson.M{"$set": updateFields}
	if err := s.Repo.UpdateWithDocument(userRec.ID, updateDoc); err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to update user password", zap.Error(err))
		return fmt.Errorf("failed to update password")
	}

	_ = utils.DeleteAuthSession(sessionClient, sessionID)
	utils.GetLogger().Sugar().Infof("ResetPassword: Password updated for user %s", userRec.ID)
	return nil
}
