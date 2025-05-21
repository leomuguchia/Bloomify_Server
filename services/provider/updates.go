package provider

import (
	"bloomify/services/user"
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

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
		updateFields["verification.legalName"] = v
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
		existing.ServiceCatalogue.Service.ID = v
	}
	if v, ok := updates["mode"].(string); ok && v != "" {
		updateFields["serviceCatalogue.mode"] = v
		existing.ServiceCatalogue.Mode = v
	}
	if v, ok := updates["customOptions"]; ok {
		if opts, ok := v.(map[string]interface{}); ok {
			newOpts := make([]models.CustomOption, 0, len(opts))
			for key, val := range opts {
				var multiplier float64
				switch t := val.(type) {
				case float64:
					multiplier = t
				case int:
					multiplier = float64(t)
				default:
					return nil, fmt.Errorf("invalid type for custom option %s", key)
				}
				newOpts = append(newOpts, models.CustomOption{
					Option:     key,
					Multiplier: multiplier,
				})
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
					updateFields["profile.locationGeo"] = geoPoint
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

func (s *DefaultProviderService) DeleteProvider(providerID string) error {
	if err := s.Repo.Delete(providerID); err != nil {
		return fmt.Errorf("failed to delete provider with id %s: %w", providerID, err)
	}
	return nil
}

func (s *DefaultProviderService) UpdateProviderPassword(providerID, currentPassword, newPassword, currentDeviceID string) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(providerID, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("provider not found")
	}

	if len(existing.Security.PasswordHash) > 0 {
		if err := bcrypt.CompareHashAndPassword([]byte(existing.Security.PasswordHash), []byte(currentPassword)); err != nil {
			return nil, fmt.Errorf("current password is incorrect")
		}
	} else {
		utils.GetLogger().Warn("Stored password hash is empty; proceeding with password update")
	}

	if err := user.VerifyPasswordComplexity(newPassword); err != nil {
		return nil, err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new password: %w", err)
	}

	existing.Security.PasswordHash = string(newHash)
	existing.UpdatedAt = time.Now()

	var retainedDevices []models.Device
	authCache := utils.GetAuthCacheClient()
	if len(existing.Devices) > 1 {
		for _, d := range existing.Devices {
			if d.DeviceID == currentDeviceID {
				retainedDevices = append(retainedDevices, d)
			} else {
				cacheKey := utils.AuthCachePrefix + providerID + ":" + d.DeviceID
				_ = authCache.Del(context.Background(), cacheKey).Err()
			}
		}
		existing.Devices = retainedDevices
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"password_hash": existing.Security.PasswordHash,
			"updated_at":    existing.UpdatedAt,
			"devices":       existing.Devices,
		},
	}

	if err := s.Repo.UpdateWithDocument(providerID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update provider password: %w", err)
	}
	return s.Repo.GetByIDWithProjection(providerID, nil)
}
