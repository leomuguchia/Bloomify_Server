package user

import (
	"errors"
	"fmt"
	"time"

	"bloomify/database/repository"
	"bloomify/models"
	"bloomify/utils"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Token durations.
const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// CustomClaims defines our JWT claims.
type CustomClaims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// AuthService defines the interface for authentication operations.
type AuthService interface {
	RegisterUser(user models.User) (*models.User, error)
	LoginUser(email, password string) (string, string, error)
	RefreshToken(refreshToken string) (string, string, error)
}

// DefaultAuthService is our concrete implementation.
type DefaultAuthService struct {
	UserRepo repository.UserRepository
}

// RegisterUser registers a new user by validating and hashing the password.
func (s *DefaultAuthService) RegisterUser(user models.User) (*models.User, error) {
	// You might enforce password complexity rules here.
	if len(user.PasswordHash) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Hash the password.
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

// LoginUser verifies credentials and returns an access token and refresh token.
func (s *DefaultAuthService) LoginUser(email, password string) (string, string, error) {
	user, err := s.UserRepo.GetByEmail(email)
	if err != nil {
		return "", "", errors.New("invalid email or password")
	}

	// Compare provided password with stored hash.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid email or password")
	}

	accessToken, err := generateToken(user.ID, AccessTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateToken(user.ID, RefreshTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Optionally, store the refresh token in a secure store (DB, Redis, etc.) for later revocation.

	return accessToken, refreshToken, nil
}

// RefreshToken verifies a refresh token and issues new tokens.
func (s *DefaultAuthService) RefreshToken(refreshToken string) (string, string, error) {
	claims, err := parseToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Optionally check token against stored refresh tokens in DB/Redis.

	accessToken, err := generateToken(claims.UserID, AccessTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
	}
	newRefreshToken, err := generateToken(claims.UserID, RefreshTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}
	return accessToken, newRefreshToken, nil
}

// parseToken verifies and parses a JWT token.
func parseToken(tokenStr string) (*CustomClaims, error) {
	jwtSecret := utils.AppConfig.JWTSecret
	if jwtSecret == "" {
		return nil, errors.New("JWT secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token claims")
}
