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

// RegisterUser creates a new user, stores device details, generates a token, updates the token hash, and clears the Redis cache.
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

	// Attach the device details. Mark this device as the creator.
	device.LastLogin = now
	device.Creator = true
	user.Devices = []models.Device{device}

	// Persist the new user.
	if err := s.Repo.Create(&user); err != nil {
		utils.GetLogger().Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	// Generate a JWT token for the new user.
	token, err := utils.GenerateToken(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		utils.GetLogger().Error("Failed to generate auth token", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	// Store the token hash in the user record.
	user.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(&user); err != nil {
		utils.GetLogger().Error("Failed to update user with token hash", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	// Clear the Redis cache entry for this user.
	cacheKey := utils.AuthCachePrefix + user.ID
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

// RevokeUserAuthToken clears the token hash from the database and removes the corresponding Redis cache.
func (s *DefaultUserService) RevokeUserAuthToken(userID string) error {
	// Retrieve the user record.
	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		utils.GetLogger().Error("Failed to retrieve user", zap.String("userID", userID), zap.Error(err))
		return fmt.Errorf("failed to logout, please try again")
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	// Clear the token hash.
	user.TokenHash = ""
	user.UpdatedAt = time.Now()
	if err := s.Repo.Update(user); err != nil {
		utils.GetLogger().Error("Failed to revoke user auth token", zap.String("userID", userID), zap.Error(err))
		return fmt.Errorf("failed to logout, please try again")
	}

	// Clear the Redis cache entry.
	cacheKey := utils.AuthCachePrefix + userID
	authCache := utils.GetAuthCacheClient()
	if err := authCache.Del(context.Background(), cacheKey).Err(); err != nil {
		utils.GetLogger().Error("Failed to clear auth cache on logout", zap.Error(err))
	}
	return nil
}
