package routes

import (
	"bloomify/handlers"
	"bloomify/middleware"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers user endpoints.
func RegisterUserRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/users")
	{
		api.POST("/register", hb.RegisterUserHandler)
		api.POST("/login", hb.AuthenticateUserHandler)

		// Protected routes (Require Authentication)
		api.Use(middleware.JWTAuthUserMiddleware(hb.UserRepo))
		api.GET("/id/:id", hb.GetUserByIDHandler)
		api.GET("/email/:email", hb.GetUserByEmailHandler)
		api.PUT("/update/:id", hb.UpdateUserHandler)
		api.DELETE("/delete/:id", hb.DeleteUserHandler)
		api.DELETE("/revoke/:id", hb.RevokeUserAuthTokenHandler)
	}
}

// RegisterProviderRoutes registers provider management endpoints.
func RegisterProviderRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/providers")
	{
		// Public provider endpoints (registration, login, KYP)
		api.POST("/register", hb.RegisterProviderHandler)
		api.POST("/login", hb.AuthenticateProviderHandler)
		api.POST("/kyp/verify", hb.KYPVerificationHandler)

		// GET endpoints with optional authentication (if token valid, full details; otherwise, only public fields)
		public := api.Group("")
		public.GET("/id/:id", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByIDHandler)
		public.GET("/email/:email", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByEmailHandler)

		// Endpoints that modify provider data require strict authentication.
		protected := api.Group("")
		protected.Use(middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, false))
		protected.PATCH("/update/:id", hb.UpdateProviderHandler)
		protected.DELETE("/delete/:id", hb.DeleteProviderHandler)
		protected.PUT("/advance-verify/:id", hb.AdvanceVerifyProviderHandler)
		protected.DELETE("/revoke/:id", hb.RevokeProviderAuthTokenHandler)
		protected.PUT("/create-timeslots/:id", hb.SetupTimeslotsHandler)
	}
}

// RegisterAIRoutes registers AI endpoints.
func RegisterAIRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/ai")
	{
		// Protected routes (Require Authentication)
		api.Use(middleware.JWTAuthUserMiddleware(hb.UserRepo))
		api.POST("/recommend", hb.AIRecommendHandler)
		api.POST("/suggest", hb.AISuggestHandler)
		api.POST("/auto-book", hb.AutoBookHandler)
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
		bookingGroup.Use(middleware.JWTAuthUserMiddleware(hb.UserRepo))
		bookingGroup.POST("/session", hb.InitiateSession)
		bookingGroup.PUT("/session/:sessionID", hb.UpdateSession)
		bookingGroup.POST("/confirm", hb.ConfirmBooking)
		bookingGroup.DELETE("/session/:sessionID", hb.CancelSession)
	}
}

// RegisterAdminRoutes sets up endpoints for admin operations.
func RegisterAdminRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	adminGroup := r.Group("/api/admin")
	{
		adminGroup.Use(middleware.JWTAuthAdminMiddleware())
		adminGroup.GET("/users", hb.AdminHandler.GetAllUsersHandler)
		adminGroup.GET("/providers", hb.AdminHandler.GetAllProvidersHandler)
	}
}

// RegisterRoutes centralizes registration of all endpoints and middleware.
func RegisterRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	// Setup global middleware (e.g., CORS) here.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	RegisterUserRoutes(r, hb)
	RegisterProviderRoutes(r, hb)
	RegisterAIRoutes(r, hb)
	RegisterHealthRoute(r)
	RegisterBookingRoutes(r, hb)
	RegisterAdminRoutes(r, hb)
}
