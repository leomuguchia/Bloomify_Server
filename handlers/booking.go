package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/booking"

	"github.com/gin-gonic/gin"
)

// BookingHandler handles HTTP requests for booking operations.
type BookingHandler struct {
	BookingSvc booking.BookingSessionService
}

// NewBookingHandler returns a new BookingHandler instance.
func NewBookingHandler(svc booking.BookingSessionService) *BookingHandler {
	return &BookingHandler{BookingSvc: svc}
}

// InitiateSession handles POST /api/booking/session.
// It expects a JSON payload corresponding to models.ServicePlan.
func (h *BookingHandler) InitiateSession(c *gin.Context) {
	var plan models.ServicePlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "details": err.Error()})
		return
	}

	sessionID, providers, err := h.BookingSvc.InitiateSession(plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID": sessionID,
		"providers": providers,
	})
}

// UpdateSession handles PUT /api/booking/session/:sessionID.
// It expects a JSON payload with a selected provider ID.
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
// It expects a JSON payload with both a sessionID and a confirmedSlot.
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm booking", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookingResult)
}
