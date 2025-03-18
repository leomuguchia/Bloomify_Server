// File: bloomify/service/user/crud.go
package user

import (
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func (s *DefaultUserService) UpdateUser(user models.User) (*models.User, error) {
	// Build an update document explicitly.
	updateFields := bson.M{
		"updated_at": time.Now(),
	}

	// Update non-null fields.
	if user.Username != "" {
		updateFields["username"] = user.Username
	}
	if user.PhoneNumber != "" {
		updateFields["phone_number"] = user.PhoneNumber
	}
	if user.Email != "" {
		updateFields["email"] = user.Email
	}
	// Always update profile_image; if user.ProfileImage is empty, set it to nil.
	if user.ProfileImage == "" {
		updateFields["profile_image"] = nil
	} else {
		updateFields["profile_image"] = user.ProfileImage
	}

	updateDoc := bson.M{"$set": updateFields}

	// Use the custom update function.
	if err := s.Repo.UpdateWithDocument(user.ID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return the updated user object.
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
	if err := VerifyPasswordComplexity(newPassword); err != nil {
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
