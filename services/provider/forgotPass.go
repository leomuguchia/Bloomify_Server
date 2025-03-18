package provider

import (
	"context"
	"fmt"
	"time"

	"bloomify/services/user"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ResetPassword resets a provider's password via a three-state OTP-based flow.
// State 1: Called with email only → initiates OTP and returns OTPPendingError.
// State 2: Called with email and OTP (but no new password) → verifies OTP and returns NewPasswordRequiredError.
// State 3: Called with email, OTP, and new password → verifies OTP, validates and updates password.
func (s *DefaultProviderService) ResetPassword(email, providedOTP, newPassword, providedSessionID string) error {
	// 1. Retrieve the provider record by email.
	provider, err := s.Repo.GetByEmailWithProjection(email, bson.M{})
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to fetch provider", zap.Error(err))
		return fmt.Errorf("failed to reset password, please try again")
	}
	if provider == nil {
		// Avoid exposing whether the email exists.
		return fmt.Errorf("invalid email")
	}

	sessionClient := utils.GetAuthCacheClient()
	ctx := context.Background()

	// 2. Determine session ID.
	sessionID := providedSessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%s", provider.ID, "reset_password")
		// Create a new password reset session with status "pending".
		authSession := utils.AuthSession{
			UserID:        provider.ID,            // Using provider.ID
			Email:         provider.Profile.Email, // provider email from profile
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

	// 4. State 1: If no OTP and no new password are provided.
	if providedOTP == "" && newPassword == "" {
		otpCacheKey := fmt.Sprintf("otp:%s", sessionID)
		// Check if an OTP is already in cache.
		_, err := sessionClient.Get(ctx, otpCacheKey).Result()
		if err != nil {
			// OTP not set; initiate OTP for password reset.
			if err := utils.InitiateDeviceOTP(provider.ID, "reset_password", provider.Profile.PhoneNumber); err != nil {
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

	// 5. ProvidedOTP is present. Verify the provided OTP.
	if err := utils.VerifyDeviceOTPRecord(provider.ID, "reset_password", providedOTP); err != nil {
		return fmt.Errorf("OTP verification failed: %w", err)
	}
	// Mark session as verified.
	authSession.Status = "otp_verified"
	authSession.LastUpdatedAt = time.Now()
	if err := utils.SaveAuthSession(sessionClient, sessionID, *authSession); err != nil {
		return fmt.Errorf("failed to update password reset session: %w", err)
	}

	// 6. State 2: OTP is verified but new password is not provided.
	if newPassword == "" {
		return NewPasswordRequiredError{SessionID: sessionID}
	}

	// 7. State 3: New password is provided. Validate new password complexity.
	if err := user.VerifyPasswordComplexity(newPassword); err != nil {
		return err
	}

	// 8. Hash the new password (consistent with provider registration).
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to hash new password", zap.Error(err))
		return fmt.Errorf("failed to process new password")
	}

	// 9. Update the provider record with a patch document.
	updateFields := bson.M{
		"password_hash": string(newHash),
		"updated_at":    time.Now(),
	}
	updateDoc := bson.M{"$set": updateFields}
	if err := s.Repo.UpdateWithDocument(provider.ID, updateDoc); err != nil {
		utils.GetLogger().Error("ResetPassword: Failed to update provider password", zap.Error(err))
		return fmt.Errorf("failed to update password")
	}

	// 10. Clear the password reset session.
	_ = utils.DeleteAuthSession(sessionClient, sessionID)

	utils.GetLogger().Sugar().Infof("ResetPassword: Password updated for provider %s", provider.ID)
	return nil
}
