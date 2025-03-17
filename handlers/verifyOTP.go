package handlers

import (
	"bloomify/models"
	"bloomify/services/provider"
	"bloomify/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var providerService provider.ProviderService

func VerifyOTPHandler(c *gin.Context) {
	logger := utils.GetLogger()
	var req struct {
		SessionID   string `json:"sessionId" binding:"required"`
		OTP         string `json:"otp" binding:"required"`
		AccountType string `json:"accountType" binding:"required"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("VerifyOTPHandler: Failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parts := strings.Split(req.SessionID, ":")
	if len(parts) != 2 {
		logger.Error("VerifyOTPHandler: Invalid session ID format", zap.String("sessionId", req.SessionID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}
	accountID := parts[0]
	// Use deviceID from middleware if available
	deviceID, exists := c.Get("deviceID")
	if !exists {
		logger.Error("VerifyOTPHandler: Device ID not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID missing"})
		return
	}

	if err := utils.VerifyDeviceOTPRecord(accountID, deviceID.(string), req.OTP); err != nil {
		logger.Error("VerifyOTPHandler: OTP verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	sessionClient := utils.GetAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, req.SessionID)
	if err != nil {
		logger.Error("VerifyOTPHandler: Failed to retrieve auth session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve auth session"})
		return
	}
	authSession.Status = "otp_verified"
	if err := utils.SaveAuthSession(sessionClient, req.SessionID, *authSession); err != nil {
		logger.Error("VerifyOTPHandler: Failed to update auth session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update auth session"})
		return
	}

	// Build device info from middleware context
	device := models.Device{
		DeviceID: deviceID.(string),
	}
	if dName, ok := c.Get("deviceName"); ok {
		device.DeviceName = dName.(string)
	}
	if ip, ok := c.Get("deviceIP"); ok {
		device.IP = ip.(string)
	}
	if loc, ok := c.Get("deviceLocation"); ok {
		device.Location = loc.(string)
	}

	switch req.AccountType {
	case "user":
		resp, err := userService.AuthenticateUser(req.Email, req.Password, device, req.SessionID)
		if err != nil {
			logger.Error("VerifyOTPHandler: User authentication failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	case "provider":
		resp, err := providerService.AuthenticateProvider(req.Email, req.Password, device, req.SessionID)
		if err != nil {
			logger.Error("VerifyOTPHandler: Provider authentication failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	default:
		logger.Error("VerifyOTPHandler: Unknown account type", zap.String("accountType", req.AccountType))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown account type"})
	}
}
