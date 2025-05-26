package user

import (
	userRepo "bloomify/database/repository/user"
	"bloomify/models"
)

type UserService interface {
	InitiateRegistration(basicData models.UserBasicRegistrationData, device models.Device) (string, int, error)
	VerifyRegistrationOTP(sessionID string, deviceID string, providedOTP string) (int, error)
	FinalizeRegistration(sessionID string, preferences []string, emailUpdates bool) (*AuthResponse, error)
	AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error)
	UpdateUser(user models.User) (*models.User, error)
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	DeleteUser(userID string) error
	RevokeUserAuthToken(userID, deviceID string) error
	UpdateUserPassword(userID, currentPassword, newPassword, currentDeviceID string) (*models.User, error)
	GetUserDevices(userID string) ([]models.Device, error)
	SignOutOtherDevices(userID, currentDeviceID string) error
	GetAllUsers() ([]models.User, error)
	ResetPassword(email, providedOTP, newPassword, providedSessionID, currentDeviceID string) error
}

// DefaultUserService is the production implementation.
type DefaultUserService struct {
	Repo userRepo.UserRepository
}

// NewPasswordRequiredError indicates that a new password is required after OTP verification.
type NewPasswordRequiredError struct {
	SessionID string
}

// AuthResponse contains the user's ID, token, and additional details.
type AuthResponse struct {
	ID           string `json:"id"`
	Token        string `json:"token"`
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	PhoneNumber  string `json:"phoneNumber,omitempty"`
	ProfileImage string `json:"profileImage,omitempty"`
	Rating       int    `json:"rating,omitempty"`
}
