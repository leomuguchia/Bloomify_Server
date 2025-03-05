// File: bloomify/service/provider/provider.go
package provider

import (
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// AuthResponse contains only the provider's ID and the JWT token.
type AuthResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// ProviderService defines business logic for provider operations.
type ProviderService interface {
	// RegisterProvider validates the provider's registration details (which must already be KYP-verified),
	// creates a new provider record, generates a token, stores its hash, and returns the new provider's ID and token.
	RegisterProvider(provider models.Provider) (*AuthResponse, error)
	// UpdateProvider updates an existing provider's profile.
	UpdateProvider(provider models.Provider) (*models.Provider, error)
	// GetProviderByID retrieves a provider (safe view) by its unique ID.
	GetProviderByID(providerID string) (*models.Provider, error)
	// GetProviderByEmail retrieves a provider (safe view) by its email.
	GetProviderByEmail(email string) (*models.Provider, error)
	// DeleteProvider removes a provider record.
	DeleteProvider(providerID string) error
	// AuthenticateProvider verifies credentials and returns ID and token.
	AuthenticateProvider(email, password string) (*AuthResponse, error)
}

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo providerRepo.ProviderRepository
}

// RegisterProvider validates that all required fields are present and that the provider has already
// been verified via KYP (as indicated by the presence of a valid KYP verification code).
// It then creates the provider record, generates a JWT token, and returns the ID and token.
func (s *DefaultProviderService) RegisterProvider(provider models.Provider) (*AuthResponse, error) {
	// Validate required basic fields.
	if provider.Email == "" || provider.Password == "" {
		return nil, fmt.Errorf("provider email and password are required")
	}
	// Validate required registration fields.
	if provider.ProviderName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	if provider.LegalName == "" {
		return nil, fmt.Errorf("legal name is required")
	}
	if provider.Location == "" {
		return nil, fmt.Errorf("street address is required")
	}
	if provider.Latitude == 0 || provider.Longitude == 0 {
		return nil, fmt.Errorf("valid map coordinates are required")
	}
	// Ensure that KYP verification details are present.
	if provider.KYPDocument == "" {
		return nil, fmt.Errorf("KYP document reference is required")
	}
	if provider.KYPVerificationCode == "" {
		return nil, fmt.Errorf("KYP verification code is missing; please complete the verification process")
	}
	// You may further validate the format of the verification code here.

	// Check for an existing provider (using a minimal projection).
	existing, err := s.Repo.GetByEmailWithProjection(provider.Email, bson.M{"id": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing provider: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("provider with email %s already exists", provider.Email)
	}

	// Mark the provider as verified using external KYP data.
	provider.VerificationStatus = "verified"
	provider.VerificationLevel = "basic"
	provider.Verified = true

	// Hash the provided password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(provider.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	provider.PasswordHash = string(hashedPassword)
	provider.Password = "" // Clear plain-text password.

	// Generate a new unique ID and set timestamps.
	provider.ID = uuid.New().String()
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	// Persist the new provider.
	if err := s.Repo.Create(&provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Generate a JWT token for the new provider.
	token, err := utils.GenerateToken(provider.ID, provider.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Store the token hash in the provider record.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(&provider); err != nil {
		return nil, fmt.Errorf("failed to update provider with token hash: %w", err)
	}

	// Return only the provider ID and token.
	return &AuthResponse{ID: provider.ID, Token: token}, nil
}

// UpdateProvider merges allowed updates and returns the updated provider record (safe view).
func (s *DefaultProviderService) UpdateProvider(provider models.Provider) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(provider.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Merge allowed updates.
	if provider.ProviderName != "" {
		existing.ProviderName = provider.ProviderName
	}
	if provider.LegalName != "" {
		existing.LegalName = provider.LegalName
	}
	if provider.PhoneNumber != "" {
		existing.PhoneNumber = provider.PhoneNumber
	}
	if provider.Location != "" {
		existing.Location = provider.Location
	}
	if provider.ServiceType != "" {
		existing.ServiceType = provider.ServiceType
	}
	existing.UpdatedAt = time.Now()

	// Persist the updates.
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}
	return s.GetProviderByID(provider.ID)
}

// GetProviderByID returns a provider by its ID using a projection to exclude sensitive fields.
func (s *DefaultProviderService) GetProviderByID(providerID string) (*models.Provider, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	provider, err := s.Repo.GetByIDWithProjection(providerID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

// GetProviderByEmail returns a provider by email using a projection to exclude sensitive fields.
func (s *DefaultProviderService) GetProviderByEmail(email string) (*models.Provider, error) {
	projection := bson.M{"password_hash": 0, "token_hash": 0}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider by email: %w", err)
	}
	return provider, nil
}

// DeleteProvider removes a provider record by its ID.
func (s *DefaultProviderService) DeleteProvider(providerID string) error {
	if err := s.Repo.Delete(providerID); err != nil {
		return fmt.Errorf("failed to delete provider with id %s: %w", providerID, err)
	}
	return nil
}

// AuthenticateProvider verifies the provider's credentials.
// If valid, it generates a new JWT token, stores its hash, and returns an AuthResponse.
func (s *DefaultProviderService) AuthenticateProvider(email, password string) (*AuthResponse, error) {
	// Retrieve provider using minimal projection (ID, email, and password_hash).
	projection := bson.M{"password_hash": 1, "id": 1, "email": 1}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider for authentication: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider with email %s not found", email)
	}

	// Verify the provided password.
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate a new JWT token.
	token, err := utils.GenerateToken(provider.ID, provider.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Update the provider record with the new token hash.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(provider); err != nil {
		return nil, fmt.Errorf("failed to update provider with token hash: %w", err)
	}

	// Return only the provider's ID and the plain token.
	return &AuthResponse{ID: provider.ID, Token: token}, nil
}
