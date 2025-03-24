package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// JWTAuthProviderMiddleware performs provider token authentication using both provider ID and device ID.
// In strict mode (optional == false), any validation failure aborts the request.
// In unstrict mode (optional == true), failures do not abort the request;
// instead, full provider access is simply not granted.
func JWTAuthProviderMiddleware(providerRepo providerRepo.ProviderRepository, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Recover from any unexpected panic.
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("JWTAuthProviderMiddleware: panic recovered", zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  500,
				})
			}
		}()

		logger := zap.L()
		ctx := context.Background()

		// Default: assume no full provider access.
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
		if tokenString == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  0,
			})
			return
		}

		// Extract both provider ID and device ID from the token.
		providerID, tokenDeviceID, err := utils.ExtractIDsFromToken(tokenString)
		if err != nil || providerID == "" || tokenDeviceID == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token or missing provider/device ID",
				"code":  0,
			})
			return
		}

		// Retrieve device ID from context (set by DeviceDetailsMiddleware).
		ctxDeviceIDVal, exists := c.Get("deviceID")
		if !exists {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing device details in context",
				"code":  0,
			})
			return
		}
		ctxDeviceID, ok := ctxDeviceIDVal.(string)
		if !ok || ctxDeviceID == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid device details in context",
				"code":  0,
			})
			return
		}

		// Compare the device ID from the token with the one from context.
		if tokenDeviceID != ctxDeviceID {
			// In strict mode, abort; in unstrict mode, simply proceed without full access.
			logger.Error("JWTAuthProviderMiddleware: device mismatch", zap.String("tokenDeviceID", tokenDeviceID), zap.String("contextDeviceID", ctxDeviceID))
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Device mismatch",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		}

		// Compute token hash.
		computedHash := utils.HashToken(tokenString)
		if computedHash == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  0,
			})
			return
		}

		// Build composite cache key using providerID and deviceID.
		cacheKey := utils.AuthCachePrefix + providerID + ":" + tokenDeviceID

		// Retrieve the auth cache client.
		authCache := utils.GetAuthCacheClient()
		if authCache == nil {
			logger.Error("JWTAuthProviderMiddleware: auth cache client is nil")
			if !optional {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		}

		// Attempt to retrieve the token hash from Redis.
		cachedHash, err := authCache.Get(ctx, cacheKey).Result()
		if err == nil {
			// If found and valid, refresh TTL (1 hour) and grant full access.
			if cachedHash == computedHash {
				_ = authCache.Expire(ctx, cacheKey, time.Hour).Err()
				c.Set("isProviderFullAccess", true)
				c.Set("providerID", providerID)
				c.Next()
				return
			}
			logger.Error("JWTAuthProviderMiddleware: token hash mismatch in cache", zap.String("providerID", providerID))
			// In strict mode, abort; in unstrict mode, proceed without full access.
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token mismatch",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		} else if err != redis.Nil {
			logger.Error("JWTAuthProviderMiddleware: error checking auth cache", zap.Error(err))
			if !optional {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  0,
				})
				return
			}
			c.Next()
			return
		}

		// Cache miss: Query the database.
		proj := bson.M{"id": 1, "token_hash": 1}
		prov, err := providerRepo.GetByIDWithProjection(providerID, proj)
		if err != nil || prov == nil {
			logger.Error("JWTAuthProviderMiddleware: provider not found in repository", zap.String("providerID", providerID), zap.Error(err))
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

		if computedHash != prov.Security.TokenHash {
			logger.Error("JWTAuthProviderMiddleware: token hash mismatch from DB", zap.String("providerID", providerID))
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

		// Save token hash to cache with 1-hour TTL.
		if err := authCache.Set(ctx, cacheKey, computedHash, time.Hour).Err(); err != nil {
			logger.Error("JWTAuthProviderMiddleware: failed to set auth cache", zap.Error(err))
		}

		// Successful verification: grant full provider access.
		c.Set("isProviderFullAccess", true)
		c.Set("providerID", providerID)
		c.Next()
	}
}
