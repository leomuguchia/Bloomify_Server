package handlers

import (
	"net/http"

	"bloomify/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetProfileHandler returns the authenticated user's profile.
func GetProfileHandler(c *gin.Context) {
	logger := getLogger(c)

	// Assume AuthMiddleware has set "userID" in context.
	userID, exists := c.Get("userID")
	if !exists {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	profile, err := user.GetProfile(userID.(string))
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

	userID, exists := c.Get("userID")
	if !exists {
		logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var updateReq models.User
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedUser, err := user.UpdateProfile(userID.(string), updateReq)
	if err != nil {
		logger.Error("Failed to update profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
