package provider

import "fmt"

// OTPPendingError indicates that OTP verification is required.
type OTPPendingError struct {
	SessionID string
}

func (e OTPPendingError) Error() string {
	return fmt.Sprintf("OTP verification required. SessionID: %s", e.SessionID)
}

// NewPasswordRequiredError indicates that a new password is required after OTP verification.
type NewPasswordRequiredError struct {
	SessionID string
}

func (e NewPasswordRequiredError) Error() string {
	return fmt.Sprintf("OTP verified. New password required. SessionID: %s", e.SessionID)
}
