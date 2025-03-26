package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func GetDeviceDetails(c *gin.Context) (deviceID string, deviceName string, err error) {
	deviceIDValue, exists := c.Get("deviceID")
	if !exists {
		return "", "", errors.New("missing device details: deviceID")
	}
	deviceNameValue, exists := c.Get("deviceName")
	if !exists {
		return "", "", errors.New("missing device details: deviceName")
	}

	deviceID, ok := deviceIDValue.(string)
	if !ok {
		return "", "", errors.New("invalid deviceID in context")
	}

	deviceName, ok = deviceNameValue.(string)
	if !ok {
		return "", "", errors.New("invalid deviceName in context")
	}

	return deviceID, deviceName, nil
}
