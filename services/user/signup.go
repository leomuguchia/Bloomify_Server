package user

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

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

// verifyPasswordComplexity checks that the password contains at least one lowercase letter,
// one uppercase letter, one digit, and one symbol.
func VerifyPasswordComplexity(pw string) error {
	var (
		hasMinLen = len(pw) >= 8
		hasUpper  = regexp.MustCompile(`[A-Z]`).MatchString(pw)
		hasLower  = regexp.MustCompile(`[a-z]`).MatchString(pw)
		hasNumber = regexp.MustCompile(`[0-9]`).MatchString(pw)
		hasSymbol = regexp.MustCompile(`[\W_]`).MatchString(pw) // non-alphanumeric
	)
	if !hasMinLen {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if !hasUpper {
		return fmt.Errorf("password must include at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must include at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must include at least one number")
	}
	if !hasSymbol {
		return fmt.Errorf("password must include at least one symbol")
	}
	return nil
}

// RegisterUser creates a new user, stores device details (with its token hash), and clears the device-specific Redis cache.
func (s *DefaultUserService) RegisterUser(user models.User, device models.Device) (*AuthResponse, error) {
	// Validate required fields.
	if user.Email == "" || user.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}
	if user.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if user.PhoneNumber == "" {
		return nil, fmt.Errorf("phone number is required")
	}

	// Verify password complexity.
	if err := VerifyPasswordComplexity(user.Password); err != nil {
		return nil, err
	}

	// Check for an existing user.
	existing, err := s.Repo.GetByEmailWithProjection(user.Email, bson.M{"id": 1})
	if err != nil {
		utils.GetLogger().Error("Failed to check for existing user", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}
	if existing != nil {
		return nil, fmt.Errorf("a user with this email already exists")
	}

	// Hash the provided password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}
	user.PasswordHash = string(hashedPassword)
	user.Password = ""

	// Generate a new unique ID and set timestamps.
	user.ID = uuid.New().String()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Prepare the device details.
	device.LastLogin = now
	device.Creator = true

	// Generate a JWT token for the new user that includes the device ID.
	token, err := utils.GenerateToken(user.ID, user.Email, device.DeviceID)
	if err != nil {
		utils.GetLogger().Error("Failed to generate auth token", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}
	// Compute the token hash and assign it to the device.
	tokenHash := utils.HashToken(token)
	device.TokenHash = tokenHash

	// Attach the device to the user.
	user.Devices = []models.Device{device}

	// Persist the new user with the device (including token hash).
	if err := s.Repo.Create(&user); err != nil {
		utils.GetLogger().Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	// Clear the Redis cache entry for this device using a composite key.
	cacheKey := utils.AuthCachePrefix + user.ID + ":" + device.DeviceID
	authCache := utils.GetAuthCacheClient()
	_ = authCache.Del(context.Background(), cacheKey)

	// Return the auth response.
	return &AuthResponse{
		ID:           user.ID,
		Token:        token,
		Username:     user.Username,
		Email:        user.Email,
		PhoneNumber:  user.PhoneNumber,
		ProfileImage: user.ProfileImage,
		Rating:       user.Rating,
	}, nil
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
