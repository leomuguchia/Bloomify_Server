package handlers

import (
	"bloomify/models"
	"net/http"

	"github.com/gin-gonic/gin"
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
	u := models.User{
		ID: userID,
		SafetySettings: models.SafetySettings{
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
