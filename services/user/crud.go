package user

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// GetUserByID retrieves a user by ID, excluding sensitive fields.
func (s *DefaultUserService) GetUserByID(userID string) (*models.User, error) {
	projection := bson.M{"passwordHash": 0, "tokenHash": 0}
	user, err := s.Repo.GetByIDWithProjection(userID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email, excluding sensitive fields.
func (s *DefaultUserService) GetUserByEmail(email string) (*models.User, error) {
	projection := bson.M{"passwordHash": 0, "tokenHash": 0}
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

	// Build update document to patch only devices and updatedAt.
	now := time.Now()
	updateDoc := bson.M{
		"devices":   user.Devices,
		"updatedAt": now,
	}

	// Update the user record using UpdateWithDocument.
	if err := s.Repo.UpdateSetDocument(userID, updateDoc); err != nil {
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
