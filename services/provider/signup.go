package provider

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ProviderAuthResponse defines the response returned after registration/authentication.
type ProviderAuthResponse struct {
	ID           string         `json:"id"`
	Token        string         `json:"token"`
	Profile      models.Profile `json:"profile"`
	CreatedAt    time.Time      `json:"created_at"`
	ProviderType string         `json:"provider_type,omitempty"`
	ServiceType  string         `json:"service_type,omitempty"`
	Rating       float64        `json:"rating,omitempty"`
}

// RegisterProvider creates a new provider, generates a token, stores its hash,
// clears the Redis cache, and returns an enriched auth response.
func (s *DefaultProviderService) RegisterProvider(provider models.Provider) (*ProviderAuthResponse, error) {
	// Validate required basic fields.
	if provider.Profile.Email == "" || provider.Password == "" {
		return nil, fmt.Errorf("provider email and password are required")
	}
	if provider.Profile.ProviderName == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	if provider.LegalName == "" {
		return nil, fmt.Errorf("legal name is required")
	}
	if provider.Location == "" {
		return nil, fmt.Errorf("street address is required")
	}
	// Validate that location_geo is provided and has exactly two coordinates.
	if provider.LocationGeo.Type != "Point" || len(provider.LocationGeo.Coordinates) != 2 {
		return nil, fmt.Errorf("valid geo coordinates are required in location_geo field")
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
	if provider.TaxPIN != "" || len(provider.InsuranceDocs) > 0 {
		provider.Profile.AdvancedVerified = true
		provider.VerificationLevel = "advanced"
	} else {
		provider.Profile.AdvancedVerified = false
		provider.VerificationLevel = "basic"
	}

	// Use default profile image if not supplied.
	defaultProfileImage := "https://example.com/default_profile.png"
	if provider.Profile.ProfileImage == "" {
		provider.Profile.ProfileImage = defaultProfileImage
	}

	// Verify password complexity.
	if err := user.VerifyPasswordComplexity(provider.Password); err != nil {
		return nil, err
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
	existing, err := s.Repo.GetByEmailWithProjection(provider.Profile.Email, bson.M{"id": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing provider: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("provider with email %s already exists", provider.Profile.Email)
	}

	// Persist the new provider.
	if err := s.Repo.Create(&provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Generate a JWT token.
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Store the token hash in the provider record.
	provider.TokenHash = utils.HashToken(token)
	if err := s.Repo.Update(&provider); err != nil {
		return nil, fmt.Errorf("failed to update provider with token hash: %w", err)
	}

	// Clear the Redis cache entry for this provider.
	cacheKey := utils.AuthCachePrefix + provider.ID
	authCache := utils.GetAuthCacheClient()
	if err := authCache.Del(context.Background(), cacheKey).Err(); err != nil {
		zap.L().Error("Failed to clear auth cache", zap.Error(err))
	}

	// Build and return the enriched auth response.
	response := &ProviderAuthResponse{
		ID:           provider.ID,
		Token:        token,
		Profile:      provider.Profile,
		CreatedAt:    provider.CreatedAt,
		ProviderType: provider.ProviderType,
		ServiceType:  provider.ServiceType,
		Rating:       provider.Rating,
	}
	return response, nil
}

// RevokeProviderAuthToken revokes the provider's auth token by clearing the token hash,
// updating the record, and clearing the Redis cache.
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

	// Clear the Redis cache entry for this provider.
	cacheKey := utils.AuthCachePrefix + providerID
	authCache := utils.GetAuthCacheClient()
	if err := authCache.Del(context.Background(), cacheKey).Err(); err != nil {
		zap.L().Error("Failed to clear auth cache on revoke", zap.Error(err))
	}
	return nil
}
