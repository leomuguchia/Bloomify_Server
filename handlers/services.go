package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetAvailableServices handles GET /api/booking/services.
func (h *BookingHandler) GetAvailableServices(c *gin.Context) {
	region := c.Param("region")
	var regionPtr *string

	if region != "" {
		regionLower := strings.ToLower(region)
		regionPtr = &regionLower
	}

	var regionStr string
	if regionPtr != nil {
		regionStr = *regionPtr
	} else {
		regionStr = ""
	}
	services, err := h.BookingSvc.GetAvailableServices(regionStr)
	if err != nil {
		h.Logger.Error("GetAvailableServices: failed to fetch services", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to fetch services",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, services)
}

// GetServiceByID handles POST /api/booking/service-details.
func (h *BookingHandler) GetServiceByID(c *gin.Context) {
	var body struct {
		ID          string `json:"id" binding:"required"`
		CountryCode string `json:"countryCode"`
		Currency    string `json:"currency"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.Logger.Error("GetServiceByID: invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"message": err.Error(),
		})
		return
	}

	if body.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing service ID",
			"message": "you must provide a service ID in the body",
		})
		return
	}

	details, err := h.BookingSvc.GetServiceByID(body.ID, body.CountryCode, body.Currency)
	if err != nil {
		h.Logger.Error("GetServiceByID: failed to fetch service details", zap.String("serviceID", body.ID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "service not found or pricing error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, details)
}
