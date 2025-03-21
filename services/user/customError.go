package user

import "fmt"

// OTPPendingError signals that OTP initiation succeeded but verification is pending.
type OTPPendingError struct {
	SessionID string
}

func (e OTPPendingError) Error() string {
	return "OTP pending; sessionID: " + e.SessionID
}

// OTPVerifiedError signals that OTP was successfully verified and preferences are now required.
type OTPVerifiedError struct {
	SessionID string
}

func (e OTPVerifiedError) Error() string {
	return "OTP verified; please submit preferences to finalize registration, sessionID: " + e.SessionID
}

func (e NewPasswordRequiredError) Error() string {
	return fmt.Sprintf("OTP verified. New password required. SessionID: %s", e.SessionID)
}
