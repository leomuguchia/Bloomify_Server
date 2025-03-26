package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"bloomify/models"
	"bloomify/services/booking"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BookingHandler handles HTTP requests for booking operations.
type BookingHandler struct {
	BookingSvc booking.BookingSessionService
	Logger     *zap.Logger
}

// NewBookingHandler returns a new BookingHandler instance.
func NewBookingHandler(svc booking.BookingSessionService, logger *zap.Logger) *BookingHandler {
	return &BookingHandler{
		BookingSvc: svc,
		Logger:     logger,
	}
}

func (h *BookingHandler) InitiateSession(c *gin.Context) {
	var servicePlan models.ServicePlan
	if err := c.ShouldBindJSON(&servicePlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request payload",
			"details": err.Error(),
		})
		return
	}

	if servicePlan.ServiceType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "serviceType is required",
		})
		return
	}

	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}
	userID, ok := userIDValue.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid userID in context",
		})
		return
	}

	deviceID, deviceName, err := GetDeviceDetails(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	sessionID, providers, err := h.BookingSvc.InitiateSession(servicePlan, userID, deviceID, deviceName)
	if err != nil {
		var matchErr *booking.MatchError
		if errors.As(err, &matchErr) {
			c.JSON(http.StatusOK, gin.H{
				"error":     matchErr.Code,
				"details":   fmt.Sprintf("No providers available for %s in your location", servicePlan.ServiceType),
				"providers": []models.ProviderDTO{},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to initiate booking session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID": sessionID,
		"providers": providers,
	})
}

// UpdateSession handles PUT /api/booking/session/:sessionID.
func (h *BookingHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("sessionID")

	var req struct {
		SelectedProviderID string `json:"selectedProviderID" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "details": err.Error()})
		return
	}

	session, err := h.BookingSvc.UpdateSession(sessionID, req.SelectedProviderID)
	if err != nil {
		h.Logger.Error("UpdateSession: failed to update booking session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID":        session.SessionID,
		"selectedProvider": session.SelectedProvider,
		"availability":     session.Availability,
	})
}

// ConfirmBooking handles POST /api/booking/confirm.
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	var req struct {
		SessionID     string               `json:"sessionID" binding:"required"`
		ConfirmedSlot models.AvailableSlot `json:"confirmedSlot" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "details": err.Error()})
		return
	}

	bookingResult, err := h.BookingSvc.ConfirmBooking(req.SessionID, req.ConfirmedSlot)
	if err != nil {
		h.Logger.Error("ConfirmBooking: failed to confirm booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm booking", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookingResult)
}

// CancelSession handles DELETE /api/booking/session/:sessionID.
func (h *BookingHandler) CancelSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionID is required"})
		return
	}
	if err := h.BookingSvc.CancelSession(sessionID); err != nil {
		h.Logger.Error("CancelSession: failed to cancel booking session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking session", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "booking session cancelled"})
}

// GetAvailableServices handles GET /api/booking/services.
func (h *BookingHandler) GetAvailableServices(c *gin.Context) {
	services, err := h.BookingSvc.GetAvailableServices()
	if err != nil {
		h.Logger.Error("GetAvailableServices: failed to fetch services", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to fetch services",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, services)
}
