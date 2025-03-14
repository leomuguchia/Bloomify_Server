// File: bloomify/service/user/crud.go
package user

import (
	"fmt"
	"time"

	userRepo "bloomify/database/repository/user"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// UserService defines business logic for user operations.
type UserService interface {
	// RegisterUser validates the user's registration details, creates a new user record,
	RegisterUser(user models.User) (*AuthResponse, error)
	// AuthenticateUser verifies credentials and returns ID and token.
	AuthenticateUser(email, password string) (*AuthResponse, error)
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

	// Admin route
	GetAllUsers() ([]models.User, error)
}

// DefaultUserService is the production implementation.
type DefaultUserService struct {
	Repo userRepo.UserRepository
}

func (s *DefaultUserService) UpdateUser(user models.User) (*models.User, error) {
	// Retrieve the full user record by explicitly passing an empty projection.
	existing, err := s.Repo.GetByIDWithProjection(user.ID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Merge allowed updates.
	if user.Username != "" {
		existing.Username = user.Username
	}
	if user.PhoneNumber != "" {
		existing.PhoneNumber = user.PhoneNumber
	}
	if user.Email != "" {
		existing.Email = user.Email
	}
	if user.ProfileImage != "" {
		existing.ProfileImage = user.ProfileImage
	}

	// Update the timestamp.
	existing.UpdatedAt = time.Now()

	// Persist the complete updated user model.
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return the updated user object using an explicit projection that omits only password_hash.
	return s.Repo.GetByIDWithProjection(user.ID, nil)
}

func (s *DefaultUserService) UpdateUserPassword(userID, currentPassword, newPassword string) (*models.User, error) {
	// Retrieve the full user record by explicitly passing an empty projection.
	existing, err := s.Repo.GetByIDWithProjection(userID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Ensure the stored hash is non-empty.
	if len(existing.PasswordHash) == 0 {
		return nil, fmt.Errorf("stored password hash is empty")
	}

	// Compare the provided current password with the stored hash.
	if err := bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte(currentPassword)); err != nil {
		return nil, fmt.Errorf("current password is incorrect")
	}

	// Verify that the new password meets complexity requirements.
	if err := verifyPasswordComplexity(newPassword); err != nil {
		return nil, err
	}

	// Hash the new password.
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new password: %w", err)
	}

	// Patch the user record with the new password hash and update timestamp.
	existing.PasswordHash = string(newHash)
	existing.UpdatedAt = time.Now()

	// Persist the updated user record.
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update user password: %w", err)
	}

	// Return the updated user object using an explicit projection that omits only the password hash.
	return s.Repo.GetByIDWithProjection(userID, nil)
}

// GetUserByID returns a user by its ID using a projection to exclude sensitive fields.
func (s *DefaultUserService) GetUserByID(userID string) (*models.User, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	user, err := s.Repo.GetByIDWithProjection(userID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByEmail returns a user by email using a projection to exclude sensitive fields.
func (s *DefaultUserService) GetUserByEmail(email string) (*models.User, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	user, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// DeleteUser removes a user record by its ID.
func (s *DefaultUserService) DeleteUser(userID string) error {
	if err := s.Repo.Delete(userID); err != nil {
		return fmt.Errorf("failed to delete user with id %s: %w", userID, err)
	}
	return nil
}

// GetAllUsers retrieves all users for admin access, excluding sensitive fields.
func (s *DefaultUserService) GetAllUsers() ([]models.User, error) {
	users, err := s.Repo.GetAllSafe()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	return users, nil
}

func (s *DefaultUserService) UpdateUserPreferences(userID string, preferences []string) error {
	// Retrieve the user by ID.
	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Update preferences and UpdatedAt timestamp.
	user.Preferences = preferences
	user.UpdatedAt = time.Now()

	// Persist the updated user.
	if err := s.Repo.Update(user); err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}
	return nil
}
