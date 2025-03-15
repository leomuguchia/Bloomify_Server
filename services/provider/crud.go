// File: bloomify/service/provider/provider.go
package provider

import (
	"fmt"
	"time"

	"bloomify/models"

	"github.com/gin-gonic/gin"
)

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
