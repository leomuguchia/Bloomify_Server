package user

import (
	"fmt"
	"time"

	userRepo "bloomify/database/repository/user"
	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// AuthResponse contains only the user's ID and the JWT token.
type AuthResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// UserService defines business logic for user operations.
type UserService interface {
	// RegisterUser validates the user's registration details, creates a new user record,
	// generates a token, stores its hash, and returns the new user's ID and token.
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
}

// DefaultUserService is the production implementation.
type DefaultUserService struct {
	Repo userRepo.UserRepository
}

// RegisterUser validates required fields, hashes the password, sets defaults, persists the user,
// generates a JWT token, and returns the user's ID and token.
func (s *DefaultUserService) RegisterUser(user models.User) (*AuthResponse, error) {
	// Validate required fields.
	if user.Email == "" || user.Password == "" {
		return nil, fmt.Errorf("user email and password are required")
	}
	if user.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Check for an existing user (using minimal projection).
	existing, err := s.Repo.GetByEmailWithProjection(user.Email, bson.M{"id": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user with email %s already exists", user.Email)
	}

	// Hash the provided password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hashedPassword)
	user.Password = "" // Clear plain-text password.

	// Generate a new unique ID and set timestamps.
	user.ID = uuid.New().String()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Persist the new user.
	if err := s.Repo.Create(&user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate a JWT token for the new user.
	token, err := utils.GenerateToken(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Store the token hash in the user record.
	user.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(&user); err != nil {
		return nil, fmt.Errorf("failed to update user with token hash: %w", err)
	}

	// Return only the user ID and token.
	return &AuthResponse{ID: user.ID, Token: token}, nil
}

// AuthenticateUser verifies the user's credentials. If valid, it generates a new JWT token,
// updates the token hash, and returns the AuthResponse.
func (s *DefaultUserService) AuthenticateUser(email, password string) (*AuthResponse, error) {
	// Retrieve user using a minimal projection.
	projection := bson.M{"password_hash": 1, "id": 1, "email": 1}
	user, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user for authentication: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user with email %s not found", email)
	}

	// Verify the provided password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate a new JWT token.
	token, err := utils.GenerateToken(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Update the user record with the new token hash.
	user.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user with token hash: %w", err)
	}

	// Return only the user's ID and token.
	return &AuthResponse{ID: user.ID, Token: token}, nil
}

// UpdateUser merges allowed updates and returns the updated user (safe view).
func (s *DefaultUserService) UpdateUser(user models.User) (*models.User, error) {
	existing, err := s.Repo.GetByIDWithProjection(user.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Merge allowed updates. For example, allow updating Username and PhoneNumber.
	if user.Username != "" {
		existing.Username = user.Username
	}
	if user.PhoneNumber != "" {
		existing.PhoneNumber = user.PhoneNumber
	}
	existing.UpdatedAt = time.Now()

	// Persist the updates.
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return s.GetUserByID(user.ID)
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
