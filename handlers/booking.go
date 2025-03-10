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
func (h *BookingHandler) InitiateSession(c *gin.Context) {
	var req models.InitiateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "details": err.Error()})
		return
	}

	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid userID in context"})
		return
	}

	plan := req.ServicePlan
	sessionID, providers, err := h.BookingSvc.InitiateSession(plan, userID, req.DeviceID, req.UserAgent)
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

// CancelSession handles DELETE /api/booking/session/:sessionID.
func (h *BookingHandler) CancelSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionID is required"})
		return
	}
	if err := h.BookingSvc.CancelSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking session", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "booking session cancelled"})
}

// --- Global Booking Handler Setup ---
// To allow direct access to booking endpoints via package-level functions.
var globalBookingHandler *BookingHandler

// SetBookingHandler assigns the global booking handler.
// This must be called during application initialization.
func SetBookingHandler(h *BookingHandler) {
	globalBookingHandler = h
}

// InitiateSession is a package-level function that delegates to the global booking handler.
func InitiateSession(c *gin.Context) {
	if globalBookingHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "booking handler not initialized"})
		return
	}
	globalBookingHandler.InitiateSession(c)
}

// UpdateSession is a package-level function that delegates to the global booking handler.
func UpdateSession(c *gin.Context) {
	if globalBookingHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "booking handler not initialized"})
		return
	}
	globalBookingHandler.UpdateSession(c)
}

// ConfirmBooking is a package-level function that delegates to the global booking handler.
func ConfirmBooking(c *gin.Context) {
	if globalBookingHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "booking handler not initialized"})
		return
	}
	globalBookingHandler.ConfirmBooking(c)
}

// CancelSession is a package-level function that delegates to the global booking handler.
func CancelSession(c *gin.Context) {
	if globalBookingHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "booking handler not initialized"})
		return
	}
	globalBookingHandler.CancelSession(c)
}
