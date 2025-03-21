package handlers

import (
	"bloomify/models"
	"bloomify/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GetUserByIDHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	usr, err := userService.GetUserByID(id)
	if err != nil {
		logger.Error("User not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// GetUserByEmailHandler handles GET /users/email/:email.
func GetUserByEmailHandler(c *gin.Context) {
	logger := utils.GetLogger()
	email := c.Param("email")
	usr, err := userService.GetUserByEmail(email)
	if err != nil {
		logger.Error("User not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

func UpdateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")

	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reqUser.ID = id

	updatedUser, err := userService.UpdateUser(reqUser)
	if err != nil {
		logger.Error("Update error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUserHandler handles DELETE /users/delete/:id.
func DeleteUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.DeleteUser(id); err != nil {
		logger.Error("Delete error", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// UpdateUserPasswordHandler handles PUT /users/password/:id.
// It expects a JSON payload with "currentPassword" and "newPassword".
func UpdateUserPasswordHandler(c *gin.Context) {
	logger := utils.GetLogger()
	userID := c.Param("id")

	// Extract device details from context (set by DeviceDetailsMiddleware).
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID"})
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid update password request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser, err := userService.UpdateUserPassword(userID, req.CurrentPassword, req.NewPassword, deviceID.(string))
	if err != nil {
		logger.Error("Failed to update user password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
