package handlers

import (
	"bloomify/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *UserHandler) GetUserByIDHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}
	idStr, ok := id.(string)
	if !ok {
		logger.Error("Invalid user ID type", zap.Any("userID", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}
	usr, err := h.UserService.GetUserByID(idStr)
	if err != nil {
		logger.Error("User not found", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// GetUserByEmailHandler handles GET /users/email/:email.
func (h *UserHandler) GetUserByEmailHandler(c *gin.Context) {
	logger := utils.GetLogger()
	email := c.Param("email")
	usr, err := h.UserService.GetUserByEmail(email)
	if err != nil {
		logger.Error("User not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// DeleteUserHandler handles DELETE /users/delete/:id.
func (h *UserHandler) DeleteUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := h.UserService.DeleteUser(id); err != nil {
		logger.Error("Delete error", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// UpdateUserPasswordHandler handles PUT /users/password/:id.
// It expects a JSON payload with "currentPassword" and "newPassword".
func (h *UserHandler) UpdateUserPasswordHandler(c *gin.Context) {
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

	updatedUser, err := h.UserService.UpdateUserPassword(userID, req.CurrentPassword, req.NewPassword, deviceID.(string))
	if err != nil {
		logger.Error("Failed to update user password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

func (uh *UserHandler) UserLegalDocumentation(c *gin.Context) {
	sections := uh.AdminService.GetLegalSectionsFor("User")

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"version": "v1.0",
		"data":    sections,
	})
}
