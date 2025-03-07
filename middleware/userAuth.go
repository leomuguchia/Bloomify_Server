// File: middleware/jwt_auth_user.go
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
	"go.uber.org/zap"
)

const (
	authCachePrefix = "auth:"
	authCacheTTL    = 10 * time.Minute
)

// JWTAuthUserMiddleware validates the JWT token for users with Redis caching.
func JWTAuthUserMiddleware(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
		ctx := context.Background()

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Extract the ID from the token.
		userID, err := utils.ExtractIDFromToken(tokenString)
		if err != nil || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token or missing user ID"})
			return
		}

		// Compute the token hash.
		computedHash := utils.HashToken(tokenString)
		cacheKey := authCachePrefix + computedHash

		// Use the dedicated auth cache client.
		authCache := utils.GetAuthCacheClient()

		// Check Redis cache.
		if cached, err := authCache.Get(ctx, cacheKey).Result(); err == nil && cached == "1" {
			// Cache hit: Refresh TTL (sliding expiration) and skip DB lookup.
			if err := authCache.Expire(ctx, cacheKey, authCacheTTL).Err(); err != nil {
				logger.Error("Failed to refresh auth cache TTL", zap.Error(err))
			}
			c.Set("userID", userID)
			c.Next()
			return
		} else if err != nil && err != redis.Nil {
			// Log the cache error, but do not block the request.
			logger.Error("Error checking auth cache", zap.Error(err))
		}

		// Cache miss: Perform DB lookup.
		proj := bson.M{"id": 1, "token_hash": 1}
		usr, err := userRepo.GetByIDWithProjection(userID, proj)
		if err != nil || usr == nil {
			logger.Error("User not found when validating token", zap.String("userID", userID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication error!"})
			return
		}

		// Compare token hash from DB with computed hash.
		if computedHash != usr.TokenHash {
			logger.Error("Token hash mismatch", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token mismatch"})
			return
		}

		// Successful validation: Cache the token hash.
		if err := authCache.Set(ctx, cacheKey, "1", authCacheTTL).Err(); err != nil {
			logger.Error("Failed to set auth cache", zap.Error(err))
		}

		// Set the user ID in the context and continue.
		c.Set("userID", userID)
		c.Next()
	}
}
