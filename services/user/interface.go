package user

import (
	userRepo "bloomify/database/repository/user"
	"bloomify/models"
	"fmt"
)

type UserService interface {
	RegisterUser(user models.User, device models.Device) (*AuthResponse, error)
	AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error)
	UpdateUser(user models.User) (*models.User, error)
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	DeleteUser(userID string) error
	RevokeUserAuthToken(userID, deviceID string) error
	UpdateUserPreferences(userID string, preferences []string) error
	UpdateUserPassword(userID, currentPassword, newPassword, currentDeviceID string) (*models.User, error)
	GetUserDevices(userID string) ([]models.Device, error)
	SignOutOtherDevices(userID, currentDeviceID string) error
	GetAllUsers() ([]models.User, error)
	ResetPassword(email, providedOTP, newPassword, providedSessionID, currentDeviceID string) error
	VerifyResetOTP(email, providedOTP, sessionID string) error
}

// DefaultUserService is the production implementation.
type DefaultUserService struct {
	Repo userRepo.UserRepository
}

// NewPasswordRequiredError indicates that a new password is required after OTP verification.
type NewPasswordRequiredError struct {
	SessionID string
}

func (e NewPasswordRequiredError) Error() string {
	return fmt.Sprintf("OTP verified. New password required. SessionID: %s", e.SessionID)
}

// OTPPendingError indicates that OTP verification is required.
type OTPPendingError struct {
	SessionID string
}

func (e OTPPendingError) Error() string {
	return fmt.Sprintf("OTP verification required. SessionID: %s", e.SessionID)
}
