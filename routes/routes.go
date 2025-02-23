package routes

import (
	"net/http"

	"bloomify/handlers"
	"bloomify/middleware"
	"bloomify/resolvers"
	"bloomify/utils"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
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
		api.GET("/providers", handlers.GetProvidersHandler)
		api.GET("/providers/:id", handlers.GetProviderHandler)
		api.POST("/providers", handlers.CreateProviderHandler)
		api.PUT("/providers/:id", handlers.UpdateProviderHandler)
		api.DELETE("/providers/:id", handlers.DeleteProviderHandler)
	}
}

// RegisterBlockedRoutes registers blocked intervals endpoints.
func RegisterBlockedRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/providers/:id/blocked", handlers.GetBlockedIntervalsHandler)
		api.POST("/providers/:id/blocked", handlers.CreateBlockedHandler)
		api.DELETE("/providers/:id/blocked/:blockedId", handlers.DeleteBlockedHandler)
	}
}

// RegisterBookingRoutes registers legacy REST booking endpoints.
func RegisterBookingRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/providers/:id/availability", handlers.GetAvailabilityHandler)
		api.GET("/providers/:id/instant", handlers.GetInstantAvailabilityHandler)
		api.POST("/bookings", handlers.CreateBookingHandler)
		api.GET("/bookings/:id", handlers.GetBookingHandler)
		api.PUT("/bookings/:id/cancel", handlers.CancelBookingHandler)
	}
}

// RegisterMatchingRoutes registers the provider matching endpoint.
func RegisterMatchingRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/providers/match", handlers.MatchProvidersHandler)
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

// --- New GraphQL Booking Endpoint (Unified Booking Engine) ---

// graphqlBookingHandler creates a Gin handler wrapping the GraphQL server for the unified booking flow.
func graphqlBookingHandler() gin.HandlerFunc {
	srv := handler.NewDefaultServer(resolvers.NewExecutableSchema(resolvers.Config{
		Resolvers: &resolvers.Resolver{
			MatchingService: resolvers.InitMatchingService(),
			BookingService:  resolvers.InitBookingService(),
			CacheClient:     utils.GetCacheClient(),
		},
	}))
	return func(c *gin.Context) {
		srv.ServeHTTP(c.Writer, c.Request)
	}
}

// playgroundBookingHandler returns a Playground handler for the booking engine.
func playgroundBookingHandler() gin.HandlerFunc {
	h := playground.Handler("Booking GraphQL", "/api/booking")
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RegisterBookingGraphQLRoutes registers the GraphQL endpoint for the unified booking engine.
// Auth middleware is applied to ensure that only authenticated users can initiate a booking session.
func RegisterBookingGraphQLRoutes(r *gin.Engine) {
	bookingGroup := r.Group("/api/booking")
	{
		bookingGroup.Use(handlers.AuthMiddleware())
		bookingGroup.POST("", graphqlBookingHandler())
		bookingGroup.GET("", playgroundBookingHandler())
	}
}

// RegisterHealthRoute registers a health-check endpoint.
func RegisterHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
