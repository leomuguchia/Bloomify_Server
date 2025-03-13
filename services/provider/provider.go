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
type ProviderService interface {
	RegisterProvider(provider models.Provider) (*models.Provider, error)
	GetProviderByID(c *gin.Context, id string) (*models.Provider, error)
	GetProviderByEmail(c *gin.Context, email string) (*models.Provider, error)
	UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error)
	DeleteProvider(id string) error
	AuthenticateProvider(email, password string) (*models.Provider, error)
	AdvanceVerifyProvider(c *gin.Context, id string, advReq AdvanceVerifyRequest) (*models.Provider, error)
	RevokeProviderAuthToken(id string) error
	SetupTimeslots(c *gin.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error)
	GetTimeslots(c *gin.Context, providerID string) ([]models.TimeSlot, error)
	DeleteTimeslot(c *gin.Context, providerID string, timeslotID string) (*models.ProviderTimeslotDTO, error)
	GetAllProviders() ([]models.Provider, error)
}

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo providerRepo.ProviderRepository
}

func (s *DefaultProviderService) RegisterProvider(provider models.Provider) (*models.Provider, error) {
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

	// Return a minimal projection (ID and token) or the full record as needed.
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
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, 24*time.Hour)
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
// It implements patch-style updates.
func (s *DefaultProviderService) UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(id, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Merge allowed fields.
	if v, ok := updates["provider_name"].(string); ok && v != "" {
		existing.Profile.ProviderName = v
	}
	if v, ok := updates["legal_name"].(string); ok && v != "" {
		existing.LegalName = v
	}
	if v, ok := updates["phone_number"].(string); ok && v != "" {
		existing.Profile.PhoneNumber = v
	}
	// Allow updating the profile image.
	if v, ok := updates["profile_image"].(string); ok && v != "" {
		existing.Profile.ProfileImage = v
	}
	if v, ok := updates["location"].(string); ok && v != "" {
		existing.Location = v
	}
	if v, ok := updates["service_type"].(string); ok && v != "" {
		existing.ServiceType = v
	}
	// Optionally update location_geo if provided.
	if geo, ok := updates["location_geo"].(map[string]interface{}); ok {
		if t, ok := geo["type"].(string); ok && t == "Point" {
			if coords, ok := geo["coordinates"].([]interface{}); ok && len(coords) == 2 {
				var newCoords []float64
				for _, c := range coords {
					switch v := c.(type) {
					case float64:
						newCoords = append(newCoords, v)
					case int:
						newCoords = append(newCoords, float64(v))
					}
				}
				if len(newCoords) == 2 {
					existing.LocationGeo = models.GeoPoint{
						Type:        "Point",
						Coordinates: newCoords,
					}
				}
			}
		}
	}

	existing.UpdatedAt = time.Now()
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return s.GetProviderByID(c, id)
}

// DeleteProvider removes a provider record by its ID.
func (s *DefaultProviderService) DeleteProvider(providerID string) error {
	if err := s.Repo.Delete(providerID); err != nil {
		return fmt.Errorf("failed to delete provider with id %s: %w", providerID, err)
	}
	return nil
}
