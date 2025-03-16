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

// AuthenticateUserHandler handles authentication requests with device management and OTP.
// It expects the following in the request JSON:
//   - email
//   - password
//   - optionally, sessionID (if retrying after OTP has been sent)
//
// Device details must have been set in context by the DeviceDetailsMiddleware.
func AuthenticateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Bind the login request.
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required"`
		SessionID string `json:"sessionID"` // optional: provided if this is a retry after OTP initiation
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

	// Call the service layer function for authentication with device management.
	authResp, err := userService.AuthenticateUser(req.Email, req.Password, currentDevice, req.SessionID)
	if err != nil {
		// If the error is an OTP pending error, return the waiting session ID.
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

	// Successful authentication.
	c.JSON(http.StatusOK, authResp)
}

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

// RevokeUserAuthTokenHandler handles DELETE /users/revoke/:id.
func RevokeUserAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.RevokeUserAuthToken(id); err != nil {
		logger.Error("Revoke token error", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}

// UpdateUserPreferencesHandler handles PUT /users/preferences/:id.
func UpdateUserPreferencesHandler(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Preferences []string `json:"preferences" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := userService.UpdateUserPreferences(userID, req.Preferences); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// UpdateUserPasswordHandler handles PUT /users/password/:id.
// It expects a JSON payload with "currentPassword" and "newPassword".
func UpdateUserPasswordHandler(c *gin.Context) {
	logger := utils.GetLogger()
	userID := c.Param("id")

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid update password request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser, err := userService.UpdateUserPassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		logger.Error("Failed to update user password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
