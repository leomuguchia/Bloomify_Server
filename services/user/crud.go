package user

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UpdateUser updates non-null user fields using a partial update.
func (s *DefaultUserService) UpdateUser(user models.User) (*models.User, error) {
	updateFields := bson.M{
		"updated_at": time.Now(),
	}
	if user.Username != "" {
		updateFields["username"] = user.Username
	}
	if user.PhoneNumber != "" {
		updateFields["phone_number"] = user.PhoneNumber
	}
	if user.Email != "" {
		updateFields["email"] = user.Email
	}
	if user.ProfileImage == "" {
		updateFields["profile_image"] = nil
	} else {
		updateFields["profile_image"] = user.ProfileImage
	}
	updateDoc := bson.M{"$set": updateFields}

	if err := s.Repo.UpdateWithDocument(user.ID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return s.Repo.GetByIDWithProjection(user.ID, nil)
}

// UpdateUserPassword updates the user's password and logs out other devices.
func (s *DefaultUserService) UpdateUserPassword(userID, currentPassword, newPassword, currentDeviceID string) (*models.User, error) {
	existing, err := s.Repo.GetByIDWithProjection(userID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Verify current password if a hash exists.
	if len(existing.PasswordHash) > 0 {
		if err := bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte(currentPassword)); err != nil {
			return nil, fmt.Errorf("current password is incorrect")
		}
	} else {
		utils.GetLogger().Warn("Stored password hash is empty; proceeding with password update")
	}

	if err := VerifyPasswordComplexity(newPassword); err != nil {
		return nil, err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new password: %w", err)
	}

	existing.PasswordHash = string(newHash)
	existing.UpdatedAt = time.Now()

	// Retain only the current device if multiple devices exist.
	var retainedDevices []models.Device
	authCache := utils.GetAuthCacheClient()
	if len(existing.Devices) > 1 {
		for _, d := range existing.Devices {
			if d.DeviceID == currentDeviceID {
				retainedDevices = append(retainedDevices, d)
			} else {
				cacheKey := utils.AuthCachePrefix + userID + ":" + d.DeviceID
				_ = authCache.Del(context.Background(), cacheKey).Err()
			}
		}
		existing.Devices = retainedDevices
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"password_hash": existing.PasswordHash,
			"updated_at":    existing.UpdatedAt,
			"devices":       existing.Devices,
		},
	}

	if err := s.Repo.UpdateWithDocument(userID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update user password: %w", err)
	}
	return s.Repo.GetByIDWithProjection(userID, nil)
}

// GetUserByID retrieves a user by ID, excluding sensitive fields.
func (s *DefaultUserService) GetUserByID(userID string) (*models.User, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	user, err := s.Repo.GetByIDWithProjection(userID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email, excluding sensitive fields.
func (s *DefaultUserService) GetUserByEmail(email string) (*models.User, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	user, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// DeleteUser removes a user by ID.
func (s *DefaultUserService) DeleteUser(userID string) error {
	if err := s.Repo.Delete(userID); err != nil {
		return fmt.Errorf("failed to delete user with id %s: %w", userID, err)
	}
	return nil
}

func (s *DefaultUserService) RevokeUserAuthToken(userID, deviceID string) error {
	// Retrieve the user record.
	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		utils.GetLogger().Error("Failed to retrieve user", zap.String("userID", userID), zap.Error(err))
		return fmt.Errorf("failed to logout, please try again")
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Clear the token hash for the specified device.
	deviceFound := false
	for i, d := range user.Devices {
		if d.DeviceID == deviceID {
			user.Devices[i].TokenHash = ""
			deviceFound = true
			break
		}
	}
	if !deviceFound {
		return fmt.Errorf("device not found")
	}

	// Build update document to patch only devices and updated_at.
	now := time.Now()
	updateDoc := bson.M{
		"$set": bson.M{
			"devices":    user.Devices,
			"updated_at": now,
		},
	}

	// Update the user record using UpdateWithDocument.
	if err := s.Repo.UpdateWithDocument(userID, updateDoc); err != nil {
		utils.GetLogger().Error("Failed to revoke user auth token", zap.String("userID", userID), zap.Error(err))
		return fmt.Errorf("failed to logout, please try again")
	}

	// Clear the Redis cache entry using the composite key.
	cacheKey := utils.AuthCachePrefix + userID + ":" + deviceID
	authCache := utils.GetAuthCacheClient()
	if err := authCache.Del(context.Background(), cacheKey).Err(); err != nil {
		utils.GetLogger().Error("Failed to clear auth cache on logout", zap.Error(err))
	}

	return nil
}
