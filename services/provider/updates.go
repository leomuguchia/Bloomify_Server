package provider

import (
	"fmt"
	"time"

	"bloomify/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// UpdateProvider merges allowed updates and returns the updated provider record (full access view).
// It implements patch-style updates using camelCase keys.
func (s *DefaultProviderService) UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(id, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	updateFields := bson.M{}

	if v, ok := updates["providerName"].(string); ok && v != "" {
		updateFields["profile.providerName"] = v
		existing.Profile.ProviderName = v
	}
	if v, ok := updates["legalName"].(string); ok && v != "" {
		updateFields["legalName"] = v
		existing.BasicVerification.LegalName = v
	}
	if v, ok := updates["phoneNumber"].(string); ok && v != "" {
		updateFields["profile.phoneNumber"] = v
		existing.Profile.PhoneNumber = v
	}
	if v, ok := updates["profileImage"].(string); ok && v != "" {
		updateFields["profile.profileImage"] = v
		existing.Profile.ProfileImage = v
	}
	if v, ok := updates["serviceType"].(string); ok && v != "" {
		updateFields["serviceCatalogue.serviceType"] = v
		existing.ServiceCatalogue.ServiceType = v
	}
	if v, ok := updates["mode"].(string); ok && v != "" {
		updateFields["serviceCatalogue.mode"] = v
		existing.ServiceCatalogue.Mode = v
	}
	if v, ok := updates["customOptions"]; ok {
		// Expecting a map[string]interface{}; convert it to map[string]float64.
		if opts, ok := v.(map[string]interface{}); ok {
			newOpts := make(map[string]float64)
			for key, val := range opts {
				switch t := val.(type) {
				case float64:
					newOpts[key] = t
				case int:
					newOpts[key] = float64(t)
				default:
					return nil, fmt.Errorf("invalid type for custom option %s", key)
				}
			}
			updateFields["serviceCatalogue.customOptions"] = newOpts
			existing.ServiceCatalogue.CustomOptions = newOpts
		}
	}
	if geo, ok := updates["locationGeo"].(map[string]interface{}); ok {
		if t, ok := geo["type"].(string); ok && t == "Point" {
			if coords, ok := geo["coordinates"].([]interface{}); ok && len(coords) == 2 {
				var newCoords []float64
				for _, cVal := range coords {
					switch v := cVal.(type) {
					case float64:
						newCoords = append(newCoords, v)
					case int:
						newCoords = append(newCoords, float64(v))
					}
				}
				if len(newCoords) == 2 {
					geoPoint := models.GeoPoint{
						Type:        "Point",
						Coordinates: newCoords,
					}
					updateFields["locationGeo"] = geoPoint
					existing.Profile.LocationGeo = geoPoint
				}
			}
		}
	}

	updateFields["updatedAt"] = time.Now()
	existing.UpdatedAt = time.Now()

	updateDoc := bson.M{
		"$set": updateFields,
	}

	if err := s.Repo.UpdateWithDocument(existing.ID, updateDoc); err != nil {
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
