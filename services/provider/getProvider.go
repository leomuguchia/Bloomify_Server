// File: service/get_provider.go
package provider

import (
	"fmt"

	"bloomify/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *DefaultProviderService) GetProviderByID(c *gin.Context, providerID string) (*models.Provider, error) {
	fullAccess, exists := c.Get("isProviderFullAccess")
	// utils.Logger.Info("Checking isProviderFullAccess flag", zap.Any("isProviderFullAccess", fullAccess), zap.Bool("exists", exists))

	access := false
	if exists {
		if fa, ok := fullAccess.(bool); ok {
			access = fa
		}
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
		projection = bson.M{
			"id":            1,
			"provider_name": 1,
			"phone_number":  1,
			"service_type":  1,
			"location":      1,
			"verified":      1,
			"rating":        1,
			"created_at":    1,
		}
	}

	provider, err := s.Repo.GetByIDWithProjection(providerID, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

func (s *DefaultProviderService) GetProviderByEmail(c *gin.Context, email string) (*models.Provider, error) {
	fullAccess, exists := c.Get("isProviderFullAccess")
	// utils.Logger.Info("Checking isProviderFullAccess flag", zap.Any("isProviderFullAccess", fullAccess), zap.Bool("exists", exists))

	access := false
	if exists {
		if fa, ok := fullAccess.(bool); ok {
			access = fa
		}
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
		projection = bson.M{
			"id":            1,
			"provider_name": 1,
			"phone_number":  1,
			"service_type":  1,
			"location":      1,
			"verified":      1,
			"rating":        1,
			"created_at":    1,
		}
	}

	provider, err := s.Repo.GetByEmailWithProjection(email, projection)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider by email: %w", err)
	}
	return provider, nil
}
