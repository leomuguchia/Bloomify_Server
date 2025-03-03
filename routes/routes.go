package routes

import (
	"bloomify/handlers"
	"bloomify/middleware"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers authentication endpoints.
func RegisterAuthRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/auth/register", handlers.RegisterHandler)
		api.POST("/auth/login", handlers.LoginHandler)
	}
}

// RegisterUserRoutes registers user endpoints.
func RegisterUserRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/users/me", middleware.AuthMiddleware(), handlers.GetProfileHandler)
		api.PUT("/users/me", middleware.AuthMiddleware(), handlers.UpdateProfileHandler)
	}
}

// RegisterProviderRoutes registers provider management endpoints.
func RegisterProviderRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Retrieve provider details by ID.
		api.GET("/providers/:id", handlers.GetProviderByIDHandler)
		// Retrieve provider details by email.
		api.GET("/providers/email/:email", handlers.GetProviderByEmailHandler)
		// Register a new provider.
		api.POST("/providers", handlers.RegisterProviderHandler)
		// Update provider details.
		api.PUT("/providers/:id", handlers.UpdateProviderHandler)
		// Delete a provider.
		api.DELETE("/providers/:id", handlers.DeleteProviderHandler)
		// Authenticate a provider.
		api.POST("/providers/authenticate", handlers.AuthenticateProviderHandler)
	}
}

// RegisterAIRoutes registers AI endpoints.
func RegisterAIRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/ai/recommend", handlers.AIRecommendHandler)
		api.POST("/ai/suggest", handlers.AISuggestHandler)
		api.POST("/ai/auto-book", handlers.AutoBookHandler)
	}
}

// RegisterHealthRoute registers a health-check endpoint.
func RegisterHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// RegisterBookingRoutes sets up the endpoints for the unified booking engine.
func RegisterBookingRoutes(r *gin.Engine, bookingHandler *handlers.BookingHandler) {
	bookingGroup := r.Group("/api/booking")
	{
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

// RegisterRoutes centralizes registration of all endpoints and middleware.
func RegisterRoutes(r *gin.Engine) {
	// Setup global middleware (e.g., CORS) here.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Call all the route registration functions.
	RegisterAuthRoutes(r)
	RegisterUserRoutes(r)
	RegisterProviderRoutes(r)
	RegisterAIRoutes(r)
	RegisterHealthRoute(r)
}
