// File: middleware/deviceAuth.go
package middleware

import (
	providerRepo "bloomify/database/repository/provider"
	userRepo "bloomify/database/repository/user"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func DeviceAuthMiddlewareUser(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Recover from unexpected panics in this middleware.
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("DeviceAuthMiddlewareUser: panic recovered", zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  0,
				})
			}
		}()

		// Check for deviceID in context.
		rawDeviceID, exists := c.Get("deviceID")
		if !exists || rawDeviceID == nil {
			zap.L().Error("DeviceAuthMiddlewareUser: missing deviceID in context")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details", "code": 0})
			return
		}
		deviceID, ok := rawDeviceID.(string)
		if !ok || deviceID == "" {
			zap.L().Error("DeviceAuthMiddlewareUser: invalid deviceID", zap.Any("deviceID", rawDeviceID))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID", "code": 0})
			return
		}

		// Check for userID in context.
		rawUserID, exists := c.Get("userID")
		if !exists || rawUserID == nil {
			zap.L().Error("DeviceAuthMiddlewareUser: missing userID in context")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "code": 0})
			return
		}
		userID, ok := rawUserID.(string)
		if !ok || userID == "" {
			zap.L().Error("DeviceAuthMiddlewareUser: invalid userID", zap.Any("userID", rawUserID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID", "code": 0})
			return
		}

		// Log the context values.
		zap.L().Info("DeviceAuthMiddlewareUser: received context values", zap.String("userID", userID), zap.String("deviceID", deviceID))

		projection := bson.M{"devices": 1}
		user, err := userRepo.GetByIDWithProjection(userID, projection)
		if err != nil || user == nil {
			zap.L().Error("DeviceAuthMiddlewareUser: user not found", zap.String("userID", userID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found", "code": 0})
			return
		}

		if user.Devices == nil {
			zap.L().Error("DeviceAuthMiddlewareUser: no devices registered for user", zap.String("userID", userID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No devices registered for user", "code": 0})
			return
		}

		deviceFound := false
		for _, d := range user.Devices {
			if d.DeviceID == deviceID {
				deviceFound = true
				break
			}
		}
		if !deviceFound {
			zap.L().Error("DeviceAuthMiddlewareUser: device not recognized", zap.String("deviceID", deviceID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized", "code": 0})
			return
		}

		c.Next()
	}
}

func DeviceAuthMiddlewareProvider(providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error("DeviceAuthMiddlewareProvider: panic recovered", zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  0,
				})
			}
		}()

		rawDeviceID, exists := c.Get("deviceID")
		if !exists || rawDeviceID == nil {
			zap.L().Error("DeviceAuthMiddlewareProvider: missing deviceID in context")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details", "code": 0})
			return
		}
		deviceID, ok := rawDeviceID.(string)
		if !ok || deviceID == "" {
			zap.L().Error("DeviceAuthMiddlewareProvider: invalid deviceID", zap.Any("deviceID", rawDeviceID))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID", "code": 0})
			return
		}

		rawProviderID, exists := c.Get("providerID")
		if !exists || rawProviderID == nil {
			zap.L().Error("DeviceAuthMiddlewareProvider: missing providerID in context")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated", "code": 0})
			return
		}
		providerID, ok := rawProviderID.(string)
		if !ok || providerID == "" {
			zap.L().Error("DeviceAuthMiddlewareProvider: invalid providerID", zap.Any("providerID", rawProviderID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid provider ID", "code": 0})
			return
		}

		zap.L().Info("DeviceAuthMiddlewareProvider: received context values", zap.String("providerID", providerID), zap.String("deviceID", deviceID))

		projection := bson.M{"devices": 1}
		provider, err := providerRepo.GetByIDWithProjection(providerID, projection)
		if err != nil || provider == nil {
			zap.L().Error("DeviceAuthMiddlewareProvider: provider not found", zap.String("providerID", providerID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not found", "code": 0})
			return
		}

		if provider.Devices == nil {
			zap.L().Error("DeviceAuthMiddlewareProvider: no devices registered for provider", zap.String("providerID", providerID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No devices registered for provider", "code": 0})
			return
		}

		deviceFound := false
		for _, d := range provider.Devices {
			if d.DeviceID == deviceID {
				deviceFound = true
				break
			}
		}
		if !deviceFound {
			zap.L().Error("DeviceAuthMiddlewareProvider: device not recognized", zap.String("deviceID", deviceID))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized", "code": 0})
			return
		}
		c.Next()
	}
}
