package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	userRepo "bloomify/database/repository/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
)

func JWTAuthUserMiddleware(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Recover from unexpected panics.
		defer func() {
			if r := recover(); r != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  500,
				})
			}
		}()

		ctx := context.Background()

		// Retrieve token from header.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid Authorization header",
				"code":  0,
			})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  0,
			})
			return
		}

		// Extract both user ID and device ID from the token.
		userID, tokenDeviceID, err := utils.ExtractIDsFromToken(tokenString)
		if err != nil || userID == "" || tokenDeviceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token or missing user/device ID",
				"code":  0,
			})
			return
		}

		// Retrieve device ID from context (set by DeviceDetailsMiddleware).
		ctxDeviceIDVal, exists := c.Get("deviceID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing device details in context",
				"code":  0,
			})
			return
		}
		ctxDeviceID, ok := ctxDeviceIDVal.(string)
		if !ok || ctxDeviceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid device details in context",
				"code":  0,
			})
			return
		}

		// Compare the deviceID from the token with the one set in context.
		if tokenDeviceID != ctxDeviceID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Device mismatch",
				"code":  0,
			})
			return
		}

		// Compute token hash.
		computedHash := utils.HashToken(tokenString)
		if computedHash == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  0,
			})
			return
		}

		// Build composite cache key using userID and deviceID.
		cacheKey := utils.AuthCachePrefix + userID + ":" + tokenDeviceID

		// Get the dedicated auth cache client.
		authCache := utils.GetAuthCacheClient()
		if authCache == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal authentication error",
				"code":  0,
			})
			return
		}

		// Attempt to retrieve the token hash from Redis.
		cachedHash, err := authCache.Get(ctx, cacheKey).Result()
		if err == nil {
			// If found and valid, refresh TTL (1 hour) and continue.
			if cachedHash == computedHash {
				_ = authCache.Expire(ctx, cacheKey, time.Hour).Err()
				c.Set("userID", userID)
				// No need to re-set "deviceID" as it's already in context.
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		} else if err != redis.Nil {
			// For errors other than key not found, you may choose to log and continue.
		}

		// Cache miss: Query the database.
		proj := bson.M{"id": 1, "devices": 1}
		usr, err := userRepo.GetByIDWithProjection(userID, proj)
		if err != nil || usr == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication error",
				"code":  0,
			})
			return
		}

		// Find the device with the matching deviceID in the user's devices.
		var deviceTokenHash string
		found := false
		for _, d := range usr.Devices {
			if d.DeviceID == tokenDeviceID {
				deviceTokenHash = d.TokenHash
				found = true
				break
			}
		}

		if !found || deviceTokenHash == "" || deviceTokenHash != computedHash {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		}

		// Successful DB validation: Save token hash to cache with 1-hour TTL.
		_ = authCache.Set(ctx, cacheKey, computedHash, time.Hour).Err()

		// Set userID in context and proceed.
		c.Set("userID", userID)
		c.Next()
	}
}
