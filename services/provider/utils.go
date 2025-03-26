// File: provider/utils.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// GenerateSessionID returns a new unique session ID.
func GenerateSessionID() string {
	return uuid.New().String()
}

// GenerateProviderID returns a new unique provider ID.
func GenerateProviderID() string {
	return uuid.New().String()
}

// GetAuthCacheClient returns the Redis client used for registration sessions.
// Replace with your actual Redis client configuration.
func GetAuthCacheClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

// GetLogger returns the application logger. Adjust as needed.
func GetLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

// SaveRegistrationSession saves the registration session to Redis with the specified TTL.
func SaveRegistrationSession(client *redis.Client, sessionID string, session models.ProviderRegistrationSession, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		GetLogger().Error("Failed to marshal registration session", zap.Error(err))
		return err
	}
	if err := client.Set(ctx, sessionID, data, ttl).Err(); err != nil {
		GetLogger().Error("Failed to save registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return err
	}
	return nil
}

// GetRegistrationSession retrieves the registration session from Redis by sessionID.
func GetRegistrationSession(client *redis.Client, sessionID string) (models.ProviderRegistrationSession, error) {
	var session models.ProviderRegistrationSession
	ctx := context.Background()
	data, err := client.Get(ctx, sessionID).Result()
	if err != nil {
		GetLogger().Error("Failed to get registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return session, err
	}
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		GetLogger().Error("Failed to unmarshal registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return session, err
	}
	return session, nil
}

// DeleteRegistrationSession removes the registration session from Redis.
func DeleteRegistrationSession(client *redis.Client, sessionID string) error {
	ctx := context.Background()
	if err := client.Del(ctx, sessionID).Err(); err != nil {
		GetLogger().Error("Failed to delete registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return err
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

func validateBasicRegistrationData(basicReq models.ProviderBasicRegistrationData) error {
	// Validate ProviderName.
	if basicReq.ProviderName == "" {
		return fmt.Errorf("username is required")
	}

	// Validate Email.
	if basicReq.Email == "" {
		return fmt.Errorf("email is required")
	}
	emailRegex := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	reEmail := regexp.MustCompile(emailRegex)
	if !reEmail.MatchString(basicReq.Email) {
		return fmt.Errorf("invalid email format")
	}

	// Validate Password.
	if basicReq.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(basicReq.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Validate Phone Number.
	if basicReq.PhoneNumber == "" {
		return fmt.Errorf("phone number is required")
	}

	// Validate Address.
	if basicReq.Address == "" {
		return fmt.Errorf("address is required")
	}
	addressRegex := `^\d+\s+[a-zA-Z0-9\s,.-]+$`
	reAddress := regexp.MustCompile(addressRegex)
	if !reAddress.MatchString(basicReq.Address) {
		return fmt.Errorf("invalid address format")
	}

	// Validate LocationGeo.
	if basicReq.LocationGeo.Type != "Point" || len(basicReq.LocationGeo.Coordinates) < 2 {
		return fmt.Errorf("invalid locationGeo data")
	}
	longitude := basicReq.LocationGeo.Coordinates[0]
	latitude := basicReq.LocationGeo.Coordinates[1]
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}

	return nil
}
