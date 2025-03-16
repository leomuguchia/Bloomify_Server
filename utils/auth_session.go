// File: bloomify/utils/auth_session.go
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const AuthSessionPrefix = "authSession:"

// AuthSession represents the progress of an authentication flow.
type AuthSession struct {
	UserID        string            `json:"userId"`
	Email         string            `json:"email"`
	Device        DeviceSessionInfo `json:"device"`
	Status        string            `json:"status"` // e.g., "pending", "otp_verified", "complete"
	CreatedAt     time.Time         `json:"createdAt"`
	LastUpdatedAt time.Time         `json:"lastUpdatedAt"`
	Token         string            `json:"token,omitempty"` // Final JWT token (set when complete)
	// Add other fields as needed.
}

// DeviceSessionInfo holds device details for the authentication session.
type DeviceSessionInfo struct {
	DeviceID   string `json:"deviceId"`
	DeviceName string `json:"deviceName"`
	IP         string `json:"ip"`
	Location   string `json:"location"`
}

// SaveAuthSession saves the authentication session in Redis with a TTL.
func SaveAuthSession(client *redis.Client, sessionID string, session AuthSession) error {
	session.LastUpdatedAt = time.Now()
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal auth session: %w", err)
	}
	ctx := context.Background()
	if err := client.Set(ctx, AuthSessionPrefix+sessionID, data, 10*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to save auth session: %w", err)
	}
	return nil
}

// GetAuthSession retrieves the authentication session from Redis.
func GetAuthSession(client *redis.Client, sessionID string) (*AuthSession, error) {
	ctx := context.Background()
	data, err := client.Get(ctx, AuthSessionPrefix+sessionID).Result()
	if err != nil {
		return nil, err
	}
	var session AuthSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth session: %w", err)
	}
	return &session, nil
}

// DeleteAuthSession removes an authentication session from Redis.
func DeleteAuthSession(client *redis.Client, sessionID string) error {
	ctx := context.Background()
	return client.Del(ctx, AuthSessionPrefix+sessionID).Err()
}
