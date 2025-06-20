package provider

import (
	"bloomify/services/tasks"
	"bloomify/services/user"
	"context"
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *DefaultProviderService) UpdateProvider(c context.Context, id string, updates map[string]any) (*models.Provider, error) {
	existing, err := s.Repo.GetByIDWithProjection(id, nil)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	updateFields := bson.M{}
	hasNotificationUpdates := false

	// Handle notification updates first if present
	if markRead, ok := updates["markNotificationsRead"]; ok {
		var notificationIDs []string

		// Handle both []string and []interface{} input types
		switch v := markRead.(type) {
		case []string:
			notificationIDs = v
		case []interface{}:
			for _, id := range v {
				if strID, ok := id.(string); ok {
					notificationIDs = append(notificationIDs, strID)
				}
			}
		}

		if len(notificationIDs) > 0 {
			err := s.Repo.MarkNotificationsAsRead(id, notificationIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to mark notifications as read: %w", err)
			}
			hasNotificationUpdates = true
			delete(updates, "markNotificationsRead")
		}
	}

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
	if v, ok := updates["status"].(string); ok && v != "" {
		updateFields["profile.status"] = v
		existing.Profile.Status = v
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

	if geo, ok := updates["locationGeo"].(map[string]any); ok {
		if t, ok := geo["type"].(string); ok && t == "Point" {
			if coords, ok := geo["coordinates"].([]any); ok && len(coords) == 2 {
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

	if v, ok := updates["reminders"]; ok {
		if reminderList, ok := v.([]any); ok {
			var reminders []models.Reminder

			for idx, r := range reminderList {
				rMap, ok := r.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("reminder at index %d is not a map[string]interface{}", idx)
				}

				reminder := models.Reminder{
					ID:    fmt.Sprint(rMap["id"]),
					Title: fmt.Sprint(rMap["title"]),
					Body:  fmt.Sprint(rMap["body"]),
					Sent:  false,
				}

				rawFireDate := rMap["fireDate"]
				ts, ok := rawFireDate.(string)
				if !ok || ts == "" {
					return nil, fmt.Errorf("missing or invalid fireDate for reminder at index %d: %v", idx, rawFireDate)
				}

				parsedTime, err := time.Parse(time.RFC3339, ts)
				if err != nil {
					return nil, fmt.Errorf("invalid fireDate format at index %d: %w", idx, err)
				}
				reminder.FireDate = parsedTime

				payload := models.ReminderPayload{
					ID:         id,
					ReminderID: reminder.ID,
					Title:      reminder.Title,
					Body:       reminder.Body,
					FireDate:   parsedTime.Format(time.RFC3339),
					Target:     "provider",
				}

				if s.AsynqClient == nil {
					return nil, fmt.Errorf("AsynqClient is nil – reminder task cannot be enqueued")
				}

				task, opts, err := tasks.NewReminderTask(payload, parsedTime)
				if err == nil {
					_, err := s.AsynqClient.Enqueue(task, opts...)
					if err != nil {
						utils.GetLogger().Error("Failed to enqueue reminder task",
							zap.Error(err), zap.String("reminderID", reminder.ID))
					}
				}

				reminders = append(reminders, reminder)
			}

			updateFields["reminders"] = reminders
			existing.Reminders = reminders
		} else {
			return nil, fmt.Errorf("expected reminders to be []any, got: %T", v)
		}
	}

	if v, ok := updates["fcmToken"].(string); ok && v != "" {
		updateFields["security.fcmToken"] = v
		existing.Security.FCMToken = v
	}

	updateFields["updatedAt"] = time.Now()
	existing.UpdatedAt = time.Now()

	// Validate we have actual updates (either fields or notifications)
	if len(updateFields) == 1 && !hasNotificationUpdates {
		return nil, fmt.Errorf("no valid update fields provided")
	}

	if len(updateFields) > 1 {
		if err := s.Repo.UpdateSetDocument(existing.ID, updateFields); err != nil {
			return nil, fmt.Errorf("failed to update provider: %w", err)
		}
	}

	return s.GetProviderByID(c, id, true)
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
		"passwordHash": existing.Security.PasswordHash,
		"updatedAt":    existing.UpdatedAt,
		"devices":      existing.Devices,
	}

	if err := s.Repo.UpdateSetDocument(providerID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update provider password: %w", err)
	}
	return s.Repo.GetByIDWithProjection(providerID, nil)
}
