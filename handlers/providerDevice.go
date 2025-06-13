package handlers

import (
	"net/http"

	"bloomify/services/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProviderDeviceHandler struct {
	ProviderService provider.ProviderService
}

func NewProviderDeviceHandler(providerService provider.ProviderService) *ProviderDeviceHandler {
	return &ProviderDeviceHandler{
		ProviderService: providerService,
	}
}

func (h *ProviderDeviceHandler) GetProviderDevicesHandler(c *gin.Context) {
	// Retrieve provider ID from context (set by JWTAuthProviderMiddleware)
	rawProviderID, exists := c.Get("providerID")
	if !exists || rawProviderID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Provider ID not found in context"})
		return
	}
	providerID, ok := rawProviderID.(string)
	if !ok || providerID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider ID in context"})
		return
	}

	devices, err := h.ProviderService.GetProviderDevices(providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *ProviderDeviceHandler) SignOutOtherProviderDevicesHandler(c *gin.Context) {
	// Retrieve provider ID from context (set by JWTAuthProviderMiddleware)
	rawProviderID, exists := c.Get("providerID")
	if !exists || rawProviderID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Provider ID not found in context"})
		return
	}
	providerID, ok := rawProviderID.(string)
	if !ok || providerID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider ID in context"})
		return
	}

	// Retrieve the current device ID from context (set by DeviceDetailsMiddleware)
	rawDeviceID, exists := c.Get("deviceID")
	if !exists || rawDeviceID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Device ID not found in context"})
		return
	}
	deviceID, ok := rawDeviceID.(string)
	if !ok || deviceID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid device ID in context"})
		return
	}

	err := h.ProviderService.SignOutOtherDevices(providerID, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signed out of other devices successfully"})
}

func (h *ProviderDeviceHandler) UpdateProviderFCMTokenHandler(c *gin.Context) {
	rawProviderID, exists := c.Get("providerID")
	if !exists || rawProviderID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Provider ID not found in context"})
		return
	}
	providerID, ok := rawProviderID.(string)
	if !ok || providerID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider ID"})
		return
	}

	var req struct {
		FCMToken string `json:"fcmToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	utils.Logger.Info("Updating FCM token for provider", zap.String("providerID", providerID), zap.String("newFCMToken", req.FCMToken))

	updates := map[string]interface{}{
		"fcmToken": req.FCMToken,
	}

	updatedProvider, err := h.ProviderService.UpdateProvider(c, providerID, updates)
	if err != nil {
		utils.Logger.Error("Failed to update FCM token for provider", zap.String("providerID", providerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	utils.Logger.Info("Successfully updated provider FCM token", zap.String("providerID", providerID))
	c.JSON(http.StatusOK, updatedProvider)
}
