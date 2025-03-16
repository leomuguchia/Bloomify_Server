package user

import (
	userRepo "bloomify/database/repository/user"
	"bloomify/models"
)

// UserService defines business logic for user operations.
type UserService interface {
	// RegisterUser validates the user's registration details, creates a new user record,
	RegisterUser(user models.User, device models.Device) (*AuthResponse, error)
	// AuthenticateUser verifies credentials and returns ID and token.
	AuthenticateUser(email, password string, currentDevice models.Device, providedSessionID string) (*AuthResponse, error)
	// UpdateUser updates an existing user's profile.
	UpdateUser(user models.User) (*models.User, error)
	// GetUserByID retrieves a user (safe view) by its unique ID.
	GetUserByID(userID string) (*models.User, error)
	// GetUserByEmail retrieves a user (safe view) by its email.
	GetUserByEmail(email string) (*models.User, error)
	// DeleteUser removes a user record.
	DeleteUser(userID string) error
	// RevokeUserAuthToken revokes the user's authentication token (for logout).
	RevokeUserAuthToken(userID string) error
	// Update User prefrences during registration
	UpdateUserPreferences(userID string, preferences []string) error
	// UpdateUserPassword verifies the current password and updates the user's password.
	UpdateUserPassword(userID, currentPassword, newPassword string) (*models.User, error)
	// Device management
	GetUserDevices(userID string) ([]models.Device, error)
	SignOutOtherDevices(userID, currentDeviceID string) error

	// Admin route
	GetAllUsers() ([]models.User, error)
}

// DefaultUserService is the production implementation.
type DefaultUserService struct {
	Repo userRepo.UserRepository
}
