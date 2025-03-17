// File: middleware/jwt_auth_user.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	userRepo "bloomify/database/repository/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// JWTAuthUserMiddleware validates the JWT token for users with Redis caching.
func JWTAuthUserMiddleware(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
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

		// Extract user ID from token.
		userID, err := utils.ExtractIDFromToken(tokenString)
		if err != nil || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token or missing user ID",
				"code":  0,
			})
			return
		}

		// Compute token hash.
		computedHash := utils.HashToken(tokenString)
		// Build Redis cache key using userID.
		cacheKey := utils.AuthCachePrefix + userID

		// Get the dedicated auth cache client.
		authCache := utils.GetAuthCacheClient()

		// Attempt to retrieve the token hash from Redis using the userID.
		cachedHash, err := authCache.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache exists, check if the cached token hash matches the computed token hash.
			if cachedHash == computedHash {
				// Refresh TTL (sliding expiration) and authenticate.
				if err := authCache.Expire(ctx, cacheKey, utils.AuthCacheTTL).Err(); err != nil {
					logger.Error("Failed to refresh auth cache TTL", zap.Error(err))
				}
				c.Set("userID", userID)
				c.Next()
				return
			}
			// Token hash mismatch in cache: authentication fails.
			logger.Error("Token hash mismatch in cache", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		} else if err != redis.Nil {
			// An error other than a missing key occurred.
			logger.Error("Error checking auth cache", zap.Error(err))
		}

		// Cache miss: Query the database.
		proj := bson.M{"id": 1, "token_hash": 1}
		usr, err := userRepo.GetByIDWithProjection(userID, proj)
		if err != nil || usr == nil {
			logger.Error("User not found in DB when validating token", zap.String("userID", userID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication error!",
				"code":  0,
			})
			return
		}

		// Compare token hash from the DB with the computed token hash.
		if computedHash != usr.TokenHash {
			logger.Error("Token hash mismatch from DB", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		}

		// Successful validation: Cache the token hash using userID as key.
		if err := authCache.Set(ctx, cacheKey, computedHash, utils.AuthCacheTTL).Err(); err != nil {
			logger.Error("Failed to set auth cache", zap.Error(err))
		}

		// Set the user ID in the context and continue.
		c.Set("userID", userID)
		c.Next()
	}
}
