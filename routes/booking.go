package routes

import (
	"bloomify/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterBookingRoutes sets up the endpoints for the unified booking engine.
// It expects a BookingHandler instance that implements the actual logic.
func RegisterBookingRoutes(r *gin.Engine, bookingHandler *handlers.BookingHandler) {
	bookingGroup := r.Group("/api/booking")
	{
		// Initiate a booking session.
		// POST /api/booking/session
		bookingGroup.POST("/session", bookingHandler.InitiateSession)

		// Update a booking session with a selected provider.
		// PUT /api/booking/session/:sessionID
		// Returns sessionID, providerID, and provider availability.
		bookingGroup.PUT("/session/:sessionID", bookingHandler.UpdateSession)

		// Confirm a booking.
		// POST /api/booking/confirm
		bookingGroup.POST("/confirm", bookingHandler.ConfirmBooking)
	}
}
