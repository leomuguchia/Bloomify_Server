package middleware

import (
	"context"
	"net/http"
	"strings"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
)

// JWTAuthProviderMiddleware validates the JWT token for providers with Redis caching.
func JWTAuthProviderMiddleware(providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
		ctx := context.Background()

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Extract the provider ID from the token.
		providerID, err := utils.ExtractIDFromToken(tokenString)
		if err != nil || providerID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Compute the token hash.
		computedHash := utils.HashToken(tokenString)
		cacheKey := authCachePrefix + computedHash

		// Check the authorization cache.
		authCache := utils.GetAuthCacheClient()
		if cached, err := authCache.Get(ctx, cacheKey).Result(); err == nil && cached == "1" {
			// Refresh TTL (sliding expiration) and proceed.
			if err := authCache.Expire(ctx, cacheKey, authCacheTTL).Err(); err != nil {
				logger.Error("Failed to refresh auth cache TTL", zap.Error(err))
			}
			c.Set("providerID", providerID)
			c.Next()
			return
		} else if err != nil && err != redis.Nil {
			logger.Error("Error checking auth cache", zap.Error(err))
		}

		// Cache miss: query the provider repository.
		proj := bson.M{"id": 1, "token_hash": 1}
		prov, err := providerRepo.GetByIDWithProjection(providerID, proj)
		if err != nil || prov == nil {
			logger.Error("Provider not found when validating token", zap.String("providerID", providerID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not found"})
			return
		}

		// Validate the token hash.
		if computedHash != prov.TokenHash {
			logger.Error("Token hash mismatch", zap.String("providerID", providerID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token mismatch"})
			return
		}

		// Successful validation: cache the token hash.
		if err := authCache.Set(ctx, cacheKey, "1", authCacheTTL).Err(); err != nil {
			logger.Error("Failed to set auth cache", zap.Error(err))
		}

		// Set the provider ID in context and proceed.
		c.Set("providerID", providerID)
		c.Next()
	}
}
