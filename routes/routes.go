package routes

import (
	providerRepo "bloomify/database/repository/provider"
	"bloomify/handlers"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers user endpoints.
func RegisterUserRoutes(r *gin.Engine, provRepo providerRepo.ProviderRepository, hb *handlers.HandlerBundle) {
	api := r.Group("/api")
	{
		api.POST("/users", hb.RegisterUserHandler)
		api.POST("/users/login", hb.AuthenticateUserHandler)
		api.GET("/users/:id", hb.GetUserByIDHandler)
		api.GET("/users/email/:email", hb.GetUserByEmailHandler)
		api.PUT("/users/:id", hb.UpdateUserHandler)
		api.DELETE("/users/:id", hb.DeleteUserHandler)
	}
}

// RegisterProviderRoutes registers provider management endpoints.
func RegisterProviderRoutes(r *gin.Engine, provRepo providerRepo.ProviderRepository, hb *handlers.HandlerBundle) {
	api := r.Group("/api")
	{
		api.GET("/providers/:id", hb.GetProviderByIDHandler)
		api.GET("/providers/email/:email", hb.GetProviderByEmailHandler)
		api.POST("/providers", hb.RegisterProviderHandler)
		api.PUT("/providers/:id", hb.UpdateProviderHandler)
		api.DELETE("/providers/:id", hb.DeleteProviderHandler)
		api.POST("/providers/login", hb.AuthenticateProviderHandler)
		// KYP verification endpoint.
		api.POST("/kyp/verify", hb.KYPVerificationHandler)
	}
}

// RegisterAIRoutes registers AI endpoints.
func RegisterAIRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api")
	{
		api.POST("/ai/recommend", hb.AIRecommendHandler)
		api.POST("/ai/suggest", hb.AISuggestHandler)
		api.POST("/ai/auto-book", hb.AutoBookHandler)
	}
}

// RegisterHealthRoute registers a health-check endpoint.
func RegisterHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Hi, I'm Bloomify"})
	})
}

// RegisterBookingRoutes sets up the endpoints for the booking engine.
func RegisterBookingRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	bookingGroup := r.Group("/api/booking")
	{
		bookingGroup.POST("/session", hb.InitiateSession)
		bookingGroup.PUT("/session/:sessionID", hb.UpdateSession)
		bookingGroup.POST("/confirm", hb.ConfirmBooking)
	}
}

// RegisterRoutes centralizes registration of all endpoints and middleware.
func RegisterRoutes(r *gin.Engine, provRepo providerRepo.ProviderRepository, hb *handlers.HandlerBundle) {
	// Setup global middleware (e.g., CORS) here.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	RegisterUserRoutes(r, provRepo, hb)
	RegisterProviderRoutes(r, provRepo, hb)
	RegisterAIRoutes(r, hb)
	RegisterHealthRoute(r)
	RegisterBookingRoutes(r, hb)
}
