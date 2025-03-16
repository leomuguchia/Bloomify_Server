package handlers

import (
	"bloomify/services/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	UserService user.UserService
}

func NewDeviceHandler(userService user.UserService) *DeviceHandler {
	return &DeviceHandler{UserService: userService}
}

func (h *DeviceHandler) GetUserDevicesHandler(c *gin.Context) {
	userID := c.Param("userID")

	devices, err := h.UserService.GetUserDevices(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *DeviceHandler) SignOutOtherDevicesHandler(c *gin.Context) {
	userID := c.Param("userID")

	currentDeviceID, exists := c.Get("deviceID")
	if !exists {
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
