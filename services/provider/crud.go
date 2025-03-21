// File: bloomify/service/provider/providercrud.go
package provider

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// UpdateProvider merges allowed updates and returns the updated provider record (full access view).
// It implements patch-style updates.
func (s *DefaultProviderService) UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(id, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Create a BSON update document.
	updateFields := bson.M{}

	if v, ok := updates["provider_name"].(string); ok && v != "" {
		updateFields["profile.providerName"] = v
		existing.Profile.ProviderName = v
	}
	if v, ok := updates["legal_name"].(string); ok && v != "" {
		updateFields["legalName"] = v
		existing.LegalName = v
	}
	if v, ok := updates["phone_number"].(string); ok && v != "" {
		updateFields["profile.phoneNumber"] = v
		existing.Profile.PhoneNumber = v
	}
	if v, ok := updates["profile_image"].(string); ok && v != "" {
		updateFields["profile.profileImage"] = v
		existing.Profile.ProfileImage = v
	}
	if v, ok := updates["location"].(string); ok && v != "" {
		updateFields["location"] = v
		existing.Location = v
	}
	if v, ok := updates["service_type"].(string); ok && v != "" {
		updateFields["serviceCatalogue.serviceType"] = v
		existing.ServiceCatalogue.ServiceType = v
	}
	if v, ok := updates["mode"].(string); ok && v != "" {
		updateFields["serviceCatalogue.mode"] = v
		existing.ServiceCatalogue.Mode = v
	}
	if v, ok := updates["custom_options"]; ok {
		// Expecting a map[string]interface{}
		if opts, ok := v.(map[string]interface{}); ok {
			updateFields["serviceCatalogue.customOptions"] = opts
			existing.ServiceCatalogue.CustomOptions = opts
		}
	}
	if geo, ok := updates["location_geo"].(map[string]interface{}); ok {
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
					updateFields["location_geo"] = geoPoint
					existing.LocationGeo = geoPoint
				}
			}
		}
	}

	updateFields["updated_at"] = time.Now()
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

func (s *DefaultProviderService) RevokeProviderAuthToken(providerID, deviceID string) error {
	// Retrieve the provider record.
	provider, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve provider: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("provider not found")
	}

	// Clear the token hash for the specified device.
	deviceFound := false
	for i, d := range provider.Devices {
		if d.DeviceID == deviceID {
			provider.Devices[i].TokenHash = ""
			deviceFound = true
			break
		}
	}
	if !deviceFound {
		return fmt.Errorf("device not found")
	}

	// Build update document to patch only devices and updated_at.
	updateDoc := bson.M{
		"$set": bson.M{
			"devices":    provider.Devices,
			"updated_at": time.Now(),
		},
	}

	// Update the provider record using UpdateWithDocument.
	if err := s.Repo.UpdateWithDocument(providerID, updateDoc); err != nil {
		return fmt.Errorf("failed to revoke provider auth token: %w", err)
	}

	// Clear the Redis cache entry using the composite key.
	cacheKey := utils.AuthCachePrefix + providerID + ":" + deviceID
	authCache := utils.GetAuthCacheClient()
	if err := authCache.Del(context.Background(), cacheKey).Err(); err != nil {
		zap.L().Error("Failed to clear auth cache on revoke", zap.Error(err))
	}

	return nil
}
