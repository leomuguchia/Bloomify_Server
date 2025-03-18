package user

import (
	"bloomify/utils"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ResetPassword resets a user's password via a three-state OTP-based flow.
// State 1: Called with email only → initiates OTP and returns OTPPendingError.
// State 2: Called with email and OTP (but no new password) → verifies OTP and returns NewPasswordRequiredError.
// State 3: Called with email, OTP, and new password → verifies OTP, validates and updates password.
func (s *DefaultUserService) ResetPassword(email, providedOTP, newPassword, providedSessionID string) error {
	// 1. Retrieve the user record by email.
	userRec, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to fetch user", zap.Error(err))
		return fmt.Errorf("failed to reset password, please try again")
	}
	if userRec == nil {
		// Avoid exposing whether the email exists.
		return fmt.Errorf("invalid email")
	}

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// 2. Determine session ID.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", userRec.ID, "reset_password")
		// Create a new password reset session with status "pending".
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

	// 3. Fetch the current session.
	authSession, err := utils.GetAuthSession(sessionClient, sessionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve password reset session: %w", err)
	}

	// 4. State 1: If no OTP and no new password is provided, initiate OTP.
	if providedOTP == "" && newPassword == "" {
		otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
		// Check if an OTP is already in cache.
		_, err := sessionClient.Get(ctx, otpCacheKey).Result()
		if err != nil {
			// OTP not set; initiate OTP for password reset.
			if err := utils.InitiateDeviceOTP(userRec.ID, "reset_password", userRec.PhoneNumber); err != nil {
				return fmt.Errorf("failed to initiate OTP: %w", err)
			}
			authSession.Status = "pending_otp"
			if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
				return fmt.Errorf("failed to update password reset session: %w", err)
			}
		}
		// Return OTPPendingError so that client knows an OTP has been sent.
		return OTPPendingError{SessionID: sessionID}
	}

	// 5. OTP verification: Only verify if the session isn't already verified.
	if authSession.Status != "otp_verified" {
		if err := utils.VerifyDeviceOTPRecord(userRec.ID, "reset_password", providedOTP); err != nil {
			return fmt.Errorf("OTP verification failed: %w", err)
		}
		// Mark session as verified.
		authSession.Status = "otp_verified"
		authSession.LastUpdatedAt = time.Now()
		if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
			return fmt.Errorf("failed to update password reset session: %w", err)
		}
	}

	// 6. If OTP is verified but new password is not provided (State 2), prompt for new password.
	if newPassword == "" {
		return NewPasswordRequiredError{SessionID: sessionID}
	}

	// 7. Now, new password is provided (State 3). Validate new password complexity.
	if err := VerifyPasswordComplexity(newPassword); err != nil {
		return err
	}

	// 8. Hash the new password (consistent with registration).
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to hash new password", zap.Error(err))
		return fmt.Errorf("failed to process new password")
	}

	// 9. Update the user record with a patch document.
	updateFields := bson.M{
		"password_hash": string(newHash),
		"updated_at":    time.Now(),
	}
	updateDoc := bson.M{"$set": updateFields}

	if err := s.Repo.UpdateWithDocument(userRec.ID, updateDoc); err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to update user password", zap.Error(err))
		return fmt.Errorf("failed to update password")
	}

	// 10. Clear the password reset session.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	// Log success.
	utils.GetLogger().Sugar().Infof("ResetPassword: Password updated for user %s", userRec.ID)
	return nil
}
