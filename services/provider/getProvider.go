// File: service/get_provider.go
package provider

import (
	"context"
	"fmt"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// GetProviderByID returns a provider by ID with fields based on access level.
func (s *DefaultProviderService) GetProviderByID(c context.Context, providerID string, fullAccess bool) (*models.Provider, error) {
	access := false
	if fullAccess {
		// If the context has full access, we allow it.
		access = true
	}

	var projection bson.M
	if access {
		// Full access: Return all details except sensitive auth fields.
		projection = bson.M{
			"password_hash": 0,
			"token_hash":    0,
		}
	} else {
		// Public access: Return only public details.
		// Only include the safe subset:
		projection = bson.M{
			"id":                       1,
			"profile.providerName":     1,
			"profile.providerType":     1,
			"profile.status":           1,
			"profile.advancedVerified": 1,
			"profile.profileImage":     1,
			"profile.rating":           1,
			"serviceCatalogue":         1,
		}
	}

	provider, err := s.Repo.GetByIDWithProjection(providerID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

// GetProviderByEmail returns a provider by email with fields based on access level.
func (s *DefaultProviderService) GetProviderByEmail(c context.Context, email string, fullAccess bool) (*models.Provider, error) {
	var projection bson.M
	if fullAccess {
		// Full access: Return all details except sensitive auth fields.
		projection = bson.M{
			"password_hash": 0,
			"token_hash":    0,
		}
	} else {
		// Public access: Return only public details.
		projection = bson.M{
			"id":                       1,
			"profile.providerName":     1,
			"profile.providerType":     1,
			"profile.status":           1,
			"profile.advancedVerified": 1,
			"profile.profileImage":     1,
			"profile.rating":           1,
			"serviceCatalogue":         1,
		}
	}

	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider by email: %w", err)
	}
	return provider, nil
}

// GetAllProviders retrieves all providers while excluding sensitive fields.
func (s *DefaultProviderService) GetAllProviders() ([]models.Provider, error) {
	// For public access we use a safe projection.
	projection := bson.M{
		"id":                       1,
		"profile.providerName":     1,
		"profile.providerType":     1,
		"profile.status":           1,
		"profile.advancedVerified": 1,
		"profile.profileImage":     1,
		"profile.rating":           1,
		"serviceCatalogue":         1,
	}
	providers, err := s.Repo.GetAllWithProjection(projection)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch providers: %w", err)
	}
	return providers, nil
}
