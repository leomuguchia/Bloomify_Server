package handlers

import (
	"net/http"
	"time"

	"bloomify/models"
	"bloomify/services/provider"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProviderHandler struct {
	Service provider.ProviderService
}

func NewProviderHandler(ps provider.ProviderService) *ProviderHandler {
	return &ProviderHandler{Service: ps}
}

func (h *ProviderHandler) RegisterProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()

	deviceID, exists := c.Get("deviceID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}
	deviceName, exists := c.Get("deviceName")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-Name"})
		return
	}
	deviceIP, _ := c.Get("deviceIP")
	deviceLocation, _ := c.Get("deviceLocation")

	device := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
		Creator:    true,
	}

	var reqProvider models.Provider
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	createdProvider, err := h.Service.RegisterProvider(reqProvider, device)
	if err != nil {
		logger.Error("Failed to register provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register provider"})
		return
	}

	c.JSON(http.StatusCreated, createdProvider)
}

func (h *ProviderHandler) AuthenticateProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required"`
		SessionID string `json:"sessionID"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID"})
		return
	}
	deviceName, _ := c.Get("deviceName")
	deviceIP, _ := c.Get("deviceIP")
	deviceLocation, _ := c.Get("deviceLocation")

	currentDevice := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
	}

	authResp, err := h.Service.AuthenticateProvider(req.Email, req.Password, currentDevice, req.SessionID)
	if err != nil {
		if otpErr, ok := err.(user.OTPPendingError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     otpErr.Error(),
				"sessionID": otpErr.SessionID,
			})
			return
		}
		logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

func (h *ProviderHandler) RevokeProviderAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	providerID := c.Param("id")

	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}

	if err := h.Service.RevokeProviderAuthToken(providerID, deviceID.(string)); err != nil {
		logger.Error("Failed to revoke provider auth token", zap.String("id", providerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke auth token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}
