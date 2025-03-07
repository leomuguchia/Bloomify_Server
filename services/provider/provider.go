// File: bloomify/service/provider/provider.go
package provider

import (
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// ProviderService defines the interface for provider-related operations.
// Note: The AuthenticateProvider and RegisterProvider methods now return a projection
// of models.Provider (with only ID and Token fields set) to serve as an auth response.
type ProviderService interface {
	RegisterProvider(provider models.Provider) (*models.Provider, error)
	GetProviderByID(c *gin.Context, id string) (*models.Provider, error)
	GetProviderByEmail(c *gin.Context, email string) (*models.Provider, error)
	UpdateProvider(c *gin.Context, provider models.Provider) (*models.Provider, error)
	DeleteProvider(id string) error
	AuthenticateProvider(email, password string) (*models.Provider, error)
	AdvanceVerifyProvider(c *gin.Context, id string, advReq AdvanceVerifyRequest) (*models.Provider, error)
	RevokeProviderAuthToken(id string) error
}

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo providerRepo.ProviderRepository
}

// RegisterProvider validates registration details, creates a new provider record,
// generates a JWT token, stores its hash, and returns a projection containing the provider's ID and token.
func (s *DefaultProviderService) RegisterProvider(provider models.Provider) (*models.Provider, error) {
	// Validate required basic fields.
	if provider.Email == "" || provider.Password == "" {
		return nil, fmt.Errorf("provider email and password are required")
	}
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

	// Mark the provider as KYP verified.
	provider.VerificationStatus = "verified"
	// Determine advanced verification:
	// advanced_verified is true only if extra details (TaxPIN or InsuranceDocs) are provided.
	if provider.TaxPIN != "" || len(provider.InsuranceDocs) > 0 {
		provider.AdvancedVerified = true
		provider.VerificationLevel = "advanced"
	} else {
		provider.AdvancedVerified = false
		provider.VerificationLevel = "basic"
	}

	// Hash the provided password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(provider.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	provider.PasswordHash = string(hashedPassword)
	provider.Password = ""

	// Generate a new unique ID and set timestamps.
	provider.ID = uuid.New().String()
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	// Check for an existing provider (using minimal projection).
	existing, err := s.Repo.GetByEmailWithProjection(provider.Email, bson.M{"id": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing provider: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("provider with email %s already exists", provider.Email)
	}

	// Persist the new provider.
	if err := s.Repo.Create(&provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Generate a JWT token.
	token, err := utils.GenerateToken(provider.ID, provider.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Store the token hash in the provider record.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(&provider); err != nil {
		return nil, fmt.Errorf("failed to update provider with token hash: %w", err)
	}

	// Return a projection containing only the provider's ID and token.
	return &models.Provider{
		ID:    provider.ID,
		Token: token,
	}, nil
}

// AuthenticateProvider verifies credentials, generates a new token,
// updates the token hash, and returns a projection containing the provider's ID and token.
func (s *DefaultProviderService) AuthenticateProvider(email, password string) (*models.Provider, error) {
	projection := bson.M{"password_hash": 1, "id": 1, "email": 1}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider for authentication: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider with email %s not found", email)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	token, err := utils.GenerateToken(provider.ID, provider.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}
	// Overwrite the token hash with the new token.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(provider); err != nil {
		return nil, fmt.Errorf("failed to update provider with token hash: %w", err)
	}
	// Return a projection containing only the provider's ID and token.
	return &models.Provider{
		ID:    provider.ID,
		Token: token,
	}, nil
}

// RevokeProviderAuthToken revokes the provider's auth token by clearing the token hash.
func (s *DefaultProviderService) RevokeProviderAuthToken(providerID string) error {
	// Retrieve the provider record.
	provider, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve provider: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("provider not found")
	}
	// Clear the token hash.
	provider.TokenHash = ""
	provider.UpdatedAt = time.Now()
	if err := s.Repo.Update(provider); err != nil {
		return fmt.Errorf("failed to revoke provider auth token: %w", err)
	}
	return nil
}

// UpdateProvider merges allowed updates and returns the updated provider record (full access view).
func (s *DefaultProviderService) UpdateProvider(c *gin.Context, provider models.Provider) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(provider.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

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

	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}
	// Now call GetProviderByID with the context and provider ID, requesting full access view.
	return s.GetProviderByID(c, provider.ID)
}

// DeleteProvider removes a provider record by its ID.
func (s *DefaultProviderService) DeleteProvider(providerID string) error {
	if err := s.Repo.Delete(providerID); err != nil {
		return fmt.Errorf("failed to delete provider with id %s: %w", providerID, err)
	}
	return nil
}
