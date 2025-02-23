package user

import (
	"fmt"
	"time"

	"bloomify/database/repository"
	"bloomify/models"
	"bloomify/utils"

	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the user service interface.
type Service interface {
	RegisterUser(user models.User) (*models.User, error)
	LoginUser(email, password string) (string, error)
	GetProfile(userID uint) (*models.User, error)
	UpdateProfile(userID uint, update models.User) (*models.User, error)
}

// DefaultService is our concrete implementation.
type DefaultService struct {
	UserRepo repository.UserRepository
	Logger   *zap.Logger
}

// RegisterUser registers a new user.
func (s *DefaultService) RegisterUser(user models.User) (*models.User, error) {
	// Validate password length, email format, etc. (omitted for brevity)
	if len(user.PasswordHash) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hashed)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Create(&user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// LoginUser authenticates a user and returns a JWT token if successful.
func (s *DefaultService) LoginUser(email, password string) (string, error) {
	user, err := s.UserRepo.GetByEmail(email)
	if err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	token, err := generateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, nil
}

// GetProfile returns the user profile.
func (s *DefaultService) GetProfile(userID uint) (*models.User, error) {
	user, err := s.UserRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// UpdateProfile updates the user's profile.
func (s *DefaultService) UpdateProfile(userID uint, update models.User) (*models.User, error) {
	user, err := s.UserRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	// Update fields (e.g., name, phone, etc.)
	user.Name = update.Name
	user.PhoneNumber = update.PhoneNumber
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	return user, nil
}

// generateToken creates a JWT for the user.
func generateToken(userID uint) (string, error) {
	jwtSecret := []byte(utils.AppConfig.JWTSecret)
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
