package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/booking"

	"github.com/gin-gonic/gin"
)

// BookingHandler holds the booking session service that orchestrates the booking process.
type BookingHandler struct {
	// BookingSvc is the unified booking session service.
	BookingSvc booking.BookingSessionService
}

// InitiateSession handles the initiation of a booking session.
// It reads a ServicePlan from the request body, calls BookingSvc.InitiateSession,
// and returns a JSON response containing the sessionID and matched providers.
func (h *BookingHandler) InitiateSession(c *gin.Context) {
	var plan models.ServicePlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service plan", "details": err.Error()})
		return
	}

	sessionID, matchedProviders, err := h.BookingSvc.InitiateSession(plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID": sessionID,
		"providers": matchedProviders,
	})
}

// UpdateSession handles updating an existing booking session when the user selects a provider.
// It extracts the sessionID from the URL and the selected providerID from the request body,
// calls BookingSvc.UpdateSession, and returns the updated session details (sessionID, providerID, availability).
func (h *BookingHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	var req struct {
		ProviderID string `json:"provider_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	updatedSession, err := h.BookingSvc.UpdateSession(sessionID, req.ProviderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionID":    sessionID,
		"providerID":   updatedSession.SelectedProvider,
		"availability": updatedSession.Availability,
	})
}

// ConfirmBooking handles the finalization of a booking.
// It reads the sessionID, confirmed availability slot, and additional booking request details from the request body,
// calls BookingSvc.ConfirmBooking, and returns the finalized booking record.
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	var req struct {
		SessionID      string                `json:"session_id" binding:"required"`
		ConfirmedSlot  models.AvailableSlot  `json:"confirmed_slot" binding:"required"`
		BookingRequest models.BookingRequest `json:"booking_request" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	booking, err := h.BookingSvc.ConfirmBooking(req.SessionID, req.ConfirmedSlot, req.BookingRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm booking", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, booking)
}
