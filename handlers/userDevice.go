package handlers

import (
	"bloomify/services/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserDeviceHandler struct {
	UserService user.UserService
}

func NewUserDeviceHandler(userService user.UserService) *UserDeviceHandler {
	return &UserDeviceHandler{UserService: userService}
}
func (h *UserDeviceHandler) GetUserDevicesHandler(c *gin.Context) {
	// Retrieve userID from context (set by JWTAuthUserMiddleware)
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return
	}

	devices, err := h.UserService.GetUserDevices(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *UserDeviceHandler) SignOutOtherUserDevicesHandler(c *gin.Context) {
	// Retrieve userID from context (set by JWTAuthUserMiddleware)
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return
	}

	currentDeviceID, exists := c.Get("deviceID")
	if !exists || currentDeviceID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Device ID not found in context"})
		return
	}

	err := h.UserService.SignOutOtherDevices(userID, currentDeviceID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signed out of other devices successfully"})
}
