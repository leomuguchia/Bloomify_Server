package user

import (
	"bloomify/utils"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// VerifyResetOTP verifies the OTP for a password reset request.
// It retrieves the user record by email, verifies the provided OTP
// (using the fixed "reset_password" identifier), and updates the auth session
// to mark it as "otp_verified".
func (s *DefaultUserService) VerifyResetOTP(email, providedOTP, sessionID string) error {
	// Retrieve the user record by email.
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("VerifyResetOTP: Failed to fetch user", zap.Error(err))
		return fmt.Errorf("invalid email or OTP")
	}
	if userRec == nil {
		return fmt.Errorf("invalid email or OTP")
	}

	sessionClient := utils.GetAuthCacheClient()

	// Retrieve the current password reset session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		utils.GetLogger().Error("VerifyResetOTP: Failed to retrieve password reset session", zap.Error(err))
		return fmt.Errorf("failed to retrieve password reset session: %w", err)
	}

	// Verify the provided OTP using the fixed "reset_password" identifier.
	if err := utils.VerifyDeviceOTPRecord(userRec.ID, "reset_password", providedOTP); err != nil {
		utils.GetLogger().Error("VerifyResetOTP: OTP verification failed", zap.Error(err))
		return fmt.Errorf("OTP verification failed: %w", err)
	}

	// Mark the session as OTP verified.
	authSession.Status = "otp_verified"
	authSession.LastUpdatedAt = time.Now()
	if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
		utils.GetLogger().Error("VerifyResetOTP: Failed to update password reset session", zap.Error(err))
		return fmt.Errorf("failed to update password reset session: %w", err)
	}

	return nil
}
