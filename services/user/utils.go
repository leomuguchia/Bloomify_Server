package user

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// VerifyPasswordComplexity checks that the password meets complexity requirements.
func VerifyPasswordComplexity(pw string) error {
	var (
		hasMinLen = len(pw) >= 8
		hasUpper  = regexp.MustCompile(`[A-Z]`).MatchString(pw)
		hasLower  = regexp.MustCompile(`[a-z]`).MatchString(pw)
		hasNumber = regexp.MustCompile(`[0-9]`).MatchString(pw)
		hasSymbol = regexp.MustCompile(`[\W_]`).MatchString(pw)
	)
	if !hasMinLen {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if !hasUpper {
		return fmt.Errorf("password must include at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must include at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must include at least one number")
	}
	if !hasSymbol {
		return fmt.Errorf("password must include at least one symbol")
	}
	return nil
}

// SaveUserRegistrationSession saves the user registration session to Redis with the specified TTL.
func SaveUserRegistrationSession(client *redis.Client, sessionID string, session models.UserRegistrationSession, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		utils.GetLogger().Error("Failed to marshal user registration session", zap.Error(err))
		return err
	}
	if err := client.Set(ctx, sessionID, data, ttl).Err(); err != nil {
		utils.GetLogger().Error("Failed to save user registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return err
	}
	return nil
}

// GetUserRegistrationSession retrieves the user registration session from Redis by sessionID.
func GetUserRegistrationSession(client *redis.Client, sessionID string) (models.UserRegistrationSession, error) {
	var session models.UserRegistrationSession
	ctx := context.Background()
	data, err := client.Get(ctx, sessionID).Result()
	if err != nil {
		utils.GetLogger().Error("Failed to get user registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return session, err
	}
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		utils.GetLogger().Error("Failed to unmarshal user registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return session, err
	}
	return session, nil
}

// DeleteUserRegistrationSession removes the user registration session from Redis.
func DeleteUserRegistrationSession(client *redis.Client, sessionID string) error {
	ctx := context.Background()
	if err := client.Del(ctx, sessionID).Err(); err != nil {
		utils.GetLogger().Error("Failed to delete user registration session", zap.String("sessionID", sessionID), zap.Error(err))
		return err
	}
	return nil
}
