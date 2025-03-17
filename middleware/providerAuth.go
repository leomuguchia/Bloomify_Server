// File: middleware/jwtAuthProvider.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// JWTAuthProviderMiddleware performs provider token authentication.
// If 'optional' is true, failures do not abort the request (flag remains false).
// If 'optional' is false, any failure in verifying the token causes an authentication error.
func JWTAuthProviderMiddleware(providerRepo providerRepo.ProviderRepository, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
		ctx := context.Background()

		// Default: assume public access (i.e. no full access).
		c.Set("isProviderFullAccess", false)

		// Retrieve the Authorization header.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid Authorization header",
				"code":  0,
			})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Extract provider ID from token.
		providerID, err := utils.ExtractIDFromToken(tokenString)
		if err != nil || providerID == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token or missing provider ID",
				"code":  0,
			})
			return
		}

		// Compute token hash.
		computedHash := utils.HashToken(tokenString)
		cacheKey := utils.AuthCachePrefix + providerID

		// Retrieve the auth cache client.
		authCache := utils.GetAuthCacheClient()
		if authCache != nil {
			cachedHash, err := authCache.Get(ctx, cacheKey).Result()
			if err == nil {
				if cachedHash == computedHash {
					if err := authCache.Expire(ctx, cacheKey, utils.AuthCacheTTL).Err(); err != nil {
						logger.Error("Failed to refresh auth cache TTL", zap.Error(err))
					}
					c.Set("isProviderFullAccess", true)
					c.Set("providerID", providerID)
					c.Next()
					return
				}
				logger.Error("Token hash mismatch in cache", zap.String("providerID", providerID))
				if !optional {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "Token mismatch",
						"code":  0,
					})
					return
				}
			} else if err != redis.Nil {
				logger.Error("Error checking auth cache", zap.Error(err))
				if !optional {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": "Internal server error",
						"code":  0,
					})
					return
				}
			}
		} else {
			logger.Error("Auth cache client is nil")
			if !optional {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  0,
				})
				return
			}
		}

		// Fallback to DB lookup.
		proj := bson.M{"id": 1, "token_hash": 1}
		prov, err := providerRepo.GetByIDWithProjection(providerID, proj)
		if err != nil || prov == nil {
			logger.Error("Provider not found in repository", zap.String("providerID", providerID), zap.Error(err))
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Provider not found",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		}

		if computedHash != prov.TokenHash {
			logger.Error("Token hash mismatch from DB", zap.String("providerID", providerID))
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token mismatch",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		}

		if authCache != nil {
			if err := authCache.Set(ctx, cacheKey, computedHash, utils.AuthCacheTTL).Err(); err != nil {
				logger.Error("Failed to set auth cache", zap.Error(err))
			}
		}

		c.Set("isProviderFullAccess", true)
		c.Set("providerID", providerID)
		c.Next()
	}
}
