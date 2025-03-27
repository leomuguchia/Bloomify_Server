package middleware

import (
	"context"
	"log"
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
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}

		// Extract both user ID and device ID from the token.
		userID, tokenDeviceID, err := utils.ExtractIDsFromToken(tokenString)
		if err != nil || userID == "" || tokenDeviceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}

		// Retrieve device ID from context (set by DeviceDetailsMiddleware).
		ctxDeviceIDVal, exists := c.Get("deviceID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}
		ctxDeviceID, ok := ctxDeviceIDVal.(string)
		if !ok || ctxDeviceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}

		// Compare the deviceID from the token with the one set in context.
		if tokenDeviceID != ctxDeviceID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}

		// Compute token hash.
		computedHash := utils.HashToken(tokenString)
		if computedHash == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Insufficient authorization",
				"code":  0,
			})
			return
		}

		// Build composite cache key using userID and deviceID.
		cacheKey := utils.AuthCachePrefix + userID + ":" + tokenDeviceID

		// Get the dedicated auth cache client.
		authCache := utils.GetAuthCacheClient()
		cacheEnabled := true
		if authCache == nil {
			// Instead of aborting, log and treat it as a cache miss.
			log.Printf("WARNING: Auth cache client not available. Falling back to DB lookup.")
			cacheEnabled = false
		}

		// Attempt to retrieve the token hash from Redis if cache is enabled.
		if cacheEnabled {
			cachedHash, err := authCache.Get(ctx, cacheKey).Result()
			if err == nil {
				// If found and valid, refresh TTL (1 hour) and continue.
				if cachedHash == computedHash {
					_ = authCache.Expire(ctx, cacheKey, time.Hour).Err()
					c.Set("userID", userID)
					c.Next()
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token mismatch",
					"code":  0,
				})
				return
			} else if err != redis.Nil {
				// Log any other error and proceed to DB lookup.
				log.Printf("WARNING: Error retrieving auth cache key: %v. Falling back to DB lookup.", err)
			}
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

		if cacheEnabled {
			_ = authCache.Set(ctx, cacheKey, computedHash, time.Hour).Err()
		}

		c.Set("userID", userID)
		c.Next()
	}
}
