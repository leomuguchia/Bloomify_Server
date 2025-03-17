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

func JWTAuthUserMiddleware(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Recover from any unexpected panics.
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("JWTAuthUserMiddleware: recovered from panic", zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  500,
				})
			}
		}()

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
		cacheKey := utils.AuthCachePrefix + userID

		// Get the dedicated auth cache client.
		authCache := utils.GetAuthCacheClient()
		if authCache == nil {
			logger.Error("JWTAuthUserMiddleware: auth cache client is nil")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal authentication error",
				"code":  0,
			})
			return
		}

		// Attempt to retrieve the token hash from Redis.
		cachedHash, err := authCache.Get(ctx, cacheKey).Result()
		if err == nil {
			// If cached value matches, refresh TTL and continue.
			if cachedHash == computedHash {
				if err := authCache.Expire(ctx, cacheKey, utils.AuthCacheTTL).Err(); err != nil {
					logger.Error("JWTAuthUserMiddleware: failed to refresh auth cache TTL", zap.Error(err))
				}
				c.Set("userID", userID)
				c.Next()
				return
			}
			logger.Error("JWTAuthUserMiddleware: token hash mismatch in cache", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		} else if err != redis.Nil {
			logger.Error("JWTAuthUserMiddleware: error checking auth cache", zap.Error(err))
		}

		// Cache miss: Query the database.
		proj := bson.M{"id": 1, "token_hash": 1}
		usr, err := userRepo.GetByIDWithProjection(userID, proj)
		if err != nil || usr == nil {
			logger.Error("JWTAuthUserMiddleware: user not found in DB", zap.String("userID", userID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication error!",
				"code":  0,
			})
			return
		}

		// Compare token hash from DB.
		if computedHash != usr.TokenHash {
			logger.Error("JWTAuthUserMiddleware: token hash mismatch from DB", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token mismatch",
				"code":  0,
			})
			return
		}

		// Successful validation: cache the token hash.
		if err := authCache.Set(ctx, cacheKey, computedHash, utils.AuthCacheTTL).Err(); err != nil {
			logger.Error("JWTAuthUserMiddleware: failed to set auth cache", zap.Error(err))
		}

		// Set the user ID in the context and continue.
		c.Set("userID", userID)
		c.Next()
	}
}
