package handlers

import (
	"bloomify/models"
	"bloomify/utils"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *ProviderHandler) SetupTimeslotsHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Retrieve provider ID from the context (set by JWTAuthProviderMiddleware).
	providerIDValue, exists := c.Get("providerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
		return
	}
	providerID, ok := providerIDValue.(string)
	if !ok || providerID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider ID in context"})
		return
	}

	// Bind the incoming JSON payload to SetupTimeslotsRequest.
	var req models.SetupTimeslotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid timeslot setup request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "message": err.Error()})
		return
	}

	dto, err := h.Service.SetupTimeslots(c.Request.Context(), providerID, req)
	if err != nil {
		logger.Error("Failed to set up timeslots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set up timeslots", "message": err.Error()})
		return
	}

	go func() {
		if err := h.Notification.NotifyScheduleUpdate(context.Background(), providerID, req); err != nil {
			fmt.Printf("⚠️ push notification failed: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Timeslot setup successful; provider status updated to active",
		"provider": dto,
	})
}

func (h *ProviderHandler) GetTimeslotsHandler(c *gin.Context) {
	providerIDValue, exists := c.Get("providerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
		return
	}
	providerID, _ := providerIDValue.(string)

	var body struct {
		Date string `json:"date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid date in request body"})
		return
	}

	timeslots, err := h.Service.GetTimeslots(c.Request.Context(), providerID, body.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch timeslots", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"timeslots": timeslots})
}

func (h *ProviderHandler) DeleteTimeslotHandler(c *gin.Context) {
	providerIDValue, exists := c.Get("providerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
		return
	}
	providerID, _ := providerIDValue.(string)

	timeslotID := c.Param("timeslotID")
	if timeslotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing timeslot ID in path"})
		return
	}

	var body struct {
		Date string `json:"date" binding:"required"` // Required field
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid date in request body"})
		return
	}

	dto, err := h.Service.DeleteTimeslot(c.Request.Context(), providerID, timeslotID, body.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete timeslot", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Timeslot deleted successfully",
		"provider": dto,
	})
}
func (h *ProviderHandler) VerifyBooking(c *gin.Context) {
	providerIDValue, exists := c.Get("providerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
		return
	}
	providerID, _ := providerIDValue.(string)
	bookingID := c.Param("bookingId")
	date := c.Query("date")

	if providerID == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerId and date are required"})
		return
	}

	booking, err := h.Service.VerifyBooking(c.Request.Context(), providerID, date, bookingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"valid": false, "message": err.Error()})
		return
	}

	publicInvoice := models.ToPublicInvoice(booking.Invoice)

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"invoice": publicInvoice,
		"option":  booking.CustomOption,
	})
}
