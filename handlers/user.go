package handlers

import (
	"net/http"
	"strconv"

	"bloomify/database/repository"
	"bloomify/models"
	userService "bloomify/services/user" // alias to avoid conflict with our variable name
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Global instance of the user service.
var UserService userService.Service = &userService.DefaultService{
	UserRepo: repository.NewUserRepository(), // Make sure NewUserRepository() is implemented.
	Logger:   utils.GetLogger(),              // Ensure GetLogger() returns a *zap.Logger.
}

// GetProfileHandler returns the authenticated user's profile.
func GetProfileHandler(c *gin.Context) {
	logger := getLogger(c)

	// Assume AuthMiddleware has set "userID" in context.
	uid, exists := c.Get("userID")
	if !exists {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert userID from string to uint.
	userIDStr, ok := uid.(string)
	if !ok {
		logger.Error("User ID in context is not a string")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	uid64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.Error("Failed to convert user ID to uint", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}
	userID := uint(uid64)

	profile, err := UserService.GetProfile(userID)
	if err != nil {
		logger.Error("Failed to get user profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfileHandler updates the authenticated user's profile.
func UpdateProfileHandler(c *gin.Context) {
	logger := getLogger(c)

	uid, exists := c.Get("userID")
	if !exists {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert userID from string to uint.
	userIDStr, ok := uid.(string)
	if !ok {
		logger.Error("User ID in context is not a string")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	uid64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.Error("Failed to convert user ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}
	userID := uint(uid64)

	var updateReq models.User
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedUser, err := UserService.UpdateProfile(userID, updateReq)
	if err != nil {
		logger.Error("Failed to update profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
