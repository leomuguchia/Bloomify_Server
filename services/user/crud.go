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
	logger := utils.GetLogger()
	logger.Debug("UpdateUser called", zap.Any("user", user))

	updateFields := map[string]any{
		"updatedAt": time.Now(),
	}

	if user.Username != "" {
		updateFields["username"] = user.Username
	}
	if user.Email != "" {
		updateFields["email"] = user.Email
	}
	if user.PhoneNumber != "" {
		updateFields["phoneNumber"] = user.PhoneNumber
	}
	if user.FCMToken != "" {
		updateFields["fcmToken"] = user.FCMToken
	}
	if user.ProfileImage != "" {
		updateFields["profileImage"] = user.ProfileImage
	}
	if user.Preferences != nil {
		updateFields["preferences"] = user.Preferences
	}
	if user.Devices != nil {
		updateFields["devices"] = user.Devices
	}
	if user.Rating != 0 {
		updateFields["rating"] = user.Rating
	}
	if user.ActiveBookings != nil {
		updateFields["activeBookings"] = user.ActiveBookings
	}
	if user.BookingHistory != nil {
		updateFields["bookingHistory"] = user.BookingHistory
	}
	if user.Notifications != nil {
		updateFields["notifications"] = user.Notifications
	}
	if !user.LastBookingTime.IsZero() {
		updateFields["lastBookingTime"] = user.LastBookingTime
	}
	if len(user.Location.Coordinates) > 0 {
		updateFields["location"] = user.Location
	}

	logger.Debug("UpdateUser updateFields", zap.Any("updateFields", updateFields))

	if len(updateFields) == 1 {
		logger.Warn("No updatable fields provided")
		return nil, fmt.Errorf("no updatable fields provided")
	}
	// Ensure the user ID is set for the update operation.
	if user.ID == "" {
		logger.Error("User ID is required for update")
		return nil, fmt.Errorf("user ID is required for update")
	}

	if err := s.Repo.UpdateWithDocument(user.ID, updateFields); err != nil {
		logger.Error("Failed to update user", zap.String("userID", user.ID), zap.Error(err))
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	updatedUser, err := s.Repo.GetByIDWithProjection(user.ID, nil)
	if err != nil {
		logger.Error("Failed to fetch updated user", zap.String("userID", user.ID), zap.Error(err))
		return nil, err
	}
	logger.Debug("UpdateUser success", zap.Any("updatedUser", updatedUser))
	return updatedUser, nil
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
