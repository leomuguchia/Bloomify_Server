package handlers

import (
	"bloomify/models"
	"bloomify/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Request payload for safety updates
type SafetySettingsPayload struct {
	NoShowThresholdMinutes int    `json:"noShowThresholdMinutes"`
	SafetyReminderMinutes  int    `json:"safetyReminderMinutes"`
	RequireInsured         bool   `json:"requireInsured"`
	AlertChannel           string `json:"alertChannel"`
}

func (h *UserHandler) UpdateSafetyPreferences(c *gin.Context) {
	var payload SafetySettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Get current user ID from context (JWT middleware should set this)
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Build a partial User object
	u := models.UserUpdateRequest{
		ID: &userID,
		SafetySettings: &models.SafetySettings{
			NoShowThresholdMinutes: payload.NoShowThresholdMinutes,
			SafetyReminderMinutes:  payload.SafetyReminderMinutes,
			RequireInsured:         payload.RequireInsured,
			AlertChannel:           payload.AlertChannel,
		},
	}

	updated, err := h.UserService.UpdateUser(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "user": updated})
}

func (h *UserHandler) UpdateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	idVal, exists := c.Get("userID")
	if !exists {
		logger.Error("Missing user ID in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}
	id, ok := idVal.(string)
	if !ok || id == "" {
		logger.Error("Invalid user ID in context", zap.Any("userID", idVal))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call service with DTO and user ID
	updatedUser, err := h.UserService.UpdateUser(req)
	if err != nil {
		logger.Error("Update error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedUser)
}
