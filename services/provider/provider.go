package provider

import (
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// ProviderService defines the business logic interface for provider operations.
type ProviderService interface {
	// RegisterProvider creates a new provider record with a hashed password.
	RegisterProvider(provider models.Provider) (*models.Provider, error)
	// UpdateProvider updates an existing provider's profile (excluding password).
	UpdateProvider(provider models.Provider) (*models.Provider, error)
	// GetProviderByID retrieves a provider by its unique ID.
	GetProviderByID(providerID string) (*models.Provider, error)
	// GetProviderByEmail retrieves a provider by its email.
	GetProviderByEmail(email string) (*models.Provider, error)
	// DeleteProvider removes a provider record.
	DeleteProvider(providerID string) error
	// AuthenticateProvider verifies the email and password for login.
	AuthenticateProvider(email, password string) (*models.Provider, error)
}

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo providerRepo.ProviderRepository
}

// RegisterProvider validates input, hashes the password, sets IDs and timestamps,
// and creates a new provider record. The returned provider excludes sensitive fields.
func (s *DefaultProviderService) RegisterProvider(provider models.Provider) (*models.Provider, error) {
	// Basic validation.
	if provider.Email == "" || provider.Name == "" {
		return nil, fmt.Errorf("provider email and name are required")
	}
	if provider.Password == "" {
		return nil, fmt.Errorf("provider password is required")
	}

	// Check for duplicate provider by email using a minimal projection.
	existing, err := s.Repo.GetByEmailWithProjection(provider.Email, bson.M{"id": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing provider: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("provider with email %s already exists", provider.Email)
	}

	// Hash the password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(provider.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	provider.PasswordHash = string(hashedPassword)
	// Clear the plain-text password.
	provider.Password = ""

	// Set a unique ID and timestamps.
	provider.ID = uuid.New().String()
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	// Persist the provider.
	if err := s.Repo.Create(&provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Return a version that excludes sensitive fields.
	return s.GetProviderByID(provider.ID)
}

// UpdateProvider retrieves the current provider record, merges allowed updates,
// and persists the changes. Returned data excludes sensitive fields.
func (s *DefaultProviderService) UpdateProvider(provider models.Provider) (*models.Provider, error) {
	// Retrieve the existing provider (full document needed for merging).
	existing, err := s.Repo.GetByIDWithProjection(provider.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Merge allowed updates.
	if provider.Name != "" {
		existing.Name = provider.Name
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
	// Update the last modified timestamp.
	existing.UpdatedAt = time.Now()

	// Persist the update.
	if err := s.Repo.Update(existing); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// Return updated provider with sensitive fields removed.
	return s.GetProviderByID(provider.ID)
}

// GetProviderByID retrieves a provider by its unique ID using a projection
// that excludes sensitive fields like the password hash.
func (s *DefaultProviderService) GetProviderByID(providerID string) (*models.Provider, error) {
	// Exclude sensitive fields.
	projection := bson.M{"password_hash": 0}
	provider, err := s.Repo.GetByIDWithProjection(providerID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

// GetProviderByEmail retrieves a provider by its email using a projection
// that excludes sensitive fields.
func (s *DefaultProviderService) GetProviderByEmail(email string) (*models.Provider, error) {
	projection := bson.M{"password_hash": 0}
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

// AuthenticateProvider verifies a provider's email and password.
// It uses a projection to fetch only authentication-related fields,
// compares the password hash, and then returns a full (non-sensitive) provider record.
func (s *DefaultProviderService) AuthenticateProvider(email, password string) (*models.Provider, error) {
	// Projection for authentication: only fetch minimal required fields.
	projection := bson.M{
		"password_hash": 1,
		"id":            1,
		"email":         1,
		"verified":      1,
	}
	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider for authentication: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider with email %s not found", email)
	}

	// Compare the stored hashed password with the provided password.
	if err := bcrypt.CompareHashAndPassword([]byte(provider.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Fetch and return the full (safe) provider record.
	return s.GetProviderByID(provider.ID)
}
