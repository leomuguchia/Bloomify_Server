package user

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *DefaultUserService) UpdateUser(user models.User) (*models.User, error) {
	logger := utils.GetLogger()
	logger.Debug("UpdateUser called", zap.Any("user", user))

	setFields := bson.M{
		"updatedAt": time.Now(),
	}
	pushFields := bson.M{}

	// Collect fields for $set
	if user.Username != "" {
		setFields["username"] = user.Username
	}
	if user.Email != "" {
		setFields["email"] = user.Email
	}
	if user.PhoneNumber != "" {
		setFields["phoneNumber"] = user.PhoneNumber
	}
	if user.FCMToken != "" {
		setFields["fcmToken"] = user.FCMToken
	}
	if user.ProfileImage != "" {
		setFields["profileImage"] = user.ProfileImage
	}
	if user.Preferences != nil {
		setFields["preferences"] = user.Preferences
	}
	if user.Devices != nil {
		setFields["devices"] = user.Devices
	}
	if user.Rating != 0 {
		setFields["rating"] = user.Rating
	}
	if user.ActiveBookings != nil {
		setFields["activeBookings"] = user.ActiveBookings
	}
	if user.BookingHistory != nil {
		setFields["bookingHistory"] = user.BookingHistory
	}
	if user.Notifications != nil {
		setFields["notifications"] = user.Notifications
	}
	if !user.LastBookingTime.IsZero() {
		setFields["lastBookingTime"] = user.LastBookingTime
	}
	if len(user.Location.Coordinates) > 0 {
		setFields["location"] = user.Location
	}
	if user.SafetySettings != (models.SafetySettings{}) {
		setFields["safetySettings"] = user.SafetySettings
	}

	// Collect fields for $push
	if len(user.TrustedProviders) > 0 {
		pushFields["trustedProviders"] = bson.M{"$each": user.TrustedProviders}
	}

	// Validate input
	if user.ID == "" {
		logger.Error("User ID is required for update")
		return nil, fmt.Errorf("user ID is required for update")
	}
	if len(setFields) == 1 && len(pushFields) == 0 {
		logger.Warn("No updatable fields provided")
		return nil, fmt.Errorf("no updatable fields provided")
	}

	// Apply $set
	if len(setFields) > 1 { // more than just "updatedAt"
		if err := s.Repo.UpdateSetDocument(user.ID, setFields); err != nil {
			logger.Error("Failed to apply $set update", zap.String("userID", user.ID), zap.Error(err))
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Apply $push
	if len(pushFields) > 0 {
		if err := s.Repo.UpdateAddToSetDocument(user.ID, pushFields); err != nil {
			logger.Error("Failed to apply $push update", zap.String("userID", user.ID), zap.Error(err))
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Fetch updated user
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
		"passwordHash": existing.PasswordHash,
		"updatedAt":    existing.UpdatedAt,
		"devices":      existing.Devices,
	}

	if err := s.Repo.UpdateSetDocument(userID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update user password: %w", err)
	}

	return s.Repo.GetByIDWithProjection(userID, nil)
}

// RemoveFromUser removes specific items from an array field in the user's document.
func (s *DefaultUserService) RemoveFromUser(userID, field string, values []any) (*models.User, error) {
	logger := utils.GetLogger()

	for _, val := range values {
		if err := s.Repo.PullFromArray(userID, field, val); err != nil {
			logger.Error("Failed to remove item from user array field",
				zap.String("field", field),
				zap.Any("value", val),
				zap.String("userID", userID),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to remove item from %s: %w", field, err)
		}
	}

	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		logger.Error("Failed to fetch updated user after removal", zap.String("userID", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to fetch updated user: %w", err)
	}

	return user, nil
}
