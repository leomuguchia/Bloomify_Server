package handlers

import (
	"net/http"
	"time"

	"bloomify/models"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var userService user.UserService

func SetUserService(us user.UserService) {
	userService = us
}

// RegisterUserHandler creates a new user with device details.
func RegisterUserHandler(c *gin.Context) {
	// Extract device details from context (set by DeviceDetailsMiddleware).
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

	// Build the device model.
	device := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
		Creator:    true,
	}

	// Bind the incoming JSON to a User model.
	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call the registration service with the user and device.
	authResp, err := userService.RegisterUser(reqUser, device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, authResp)
}

// AuthenticateUserHandler handles user sign‑in with device management and OTP.
// It expects JSON containing email, password, and optionally a sessionID.
func AuthenticateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Bind the login request.
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

	// Extract device details from context (set by DeviceDetailsMiddleware).
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID"})
		return
	}
	deviceName, _ := c.Get("deviceName")
	deviceIP, _ := c.Get("deviceIP")
	deviceLocation, _ := c.Get("deviceLocation")

	// Build the current device object.
	currentDevice := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
	}

	// Call the authentication service with device management.
	authResp, err := userService.AuthenticateUser(req.Email, req.Password, currentDevice, req.SessionID)
	if err != nil {
		// If OTP is pending, return that information.
		if otpErr, ok := err.(user.OTPPendingError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verification required",
				"code":      100,
				"sessionID": otpErr.SessionID,
			})
			return
		}
		logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Successful authentication.
	c.JSON(http.StatusOK, authResp)
}

// RevokeUserAuthTokenHandler handles token revocation for a user.
// It requires the user ID in the URL parameter and uses the device details from context.
func RevokeUserAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	userID := c.Param("id")

	// Extract device ID from context.
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}

	// Call the service to revoke the token for this specific device.
	if err := userService.RevokeUserAuthToken(userID, deviceID.(string)); err != nil {
		logger.Error("Revoke token error", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}
