package routes

import (
	"bloomify/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterBookingRoutes registers all endpoints for the booking engine.
func RegisterBookingRoutes(r *gin.Engine) {
	booking := r.Group("/api/booking")
	{
		booking.POST("/session", handlers.StartBookingSession)            // Phase 1: Start session
		booking.PUT("/session/:sessionID", handlers.UpdateBookingSession) // Phase 2: Update session
		booking.POST("/confirm", handlers.ConfirmBooking)                 // Phase 3: Confirm booking
	}
}
