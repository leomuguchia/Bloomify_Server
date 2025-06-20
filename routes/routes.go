package routes

import (
	"bloomify/handlers"
	"bloomify/middleware"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/users")
	api.Use(middleware.DeviceDetailsMiddleware()) // Extract device details

	{
		api.POST("/register", hb.RegisterUserHandler)
		api.POST("/login", hb.AuthenticateUserHandler)
		api.POST("/reset-password", hb.ResetPasswordHandler)
		api.GET("/legal", hb.UserLegalDocumentation)
	}

	// Protected routes
	api.Use(
		middleware.JWTAuthUserMiddleware(hb.UserRepo),
		// middleware.DeviceAuthMiddlewareUser(hb.UserRepo),
	)
	{
		api.GET("/id", hb.GetUserByIDHandler)
		api.GET("/email/:email", hb.GetUserByEmailHandler)
		api.PUT("/update/:id", hb.UpdateUserHandler)
		api.DELETE("/delete/:id", hb.DeleteUserHandler)
		api.DELETE("/revoke/:id", hb.RevokeUserAuthTokenHandler)
		api.PUT("/password/:id", hb.UpdateUserPasswordHandler)
		api.GET("/devices", hb.GetUserDevicesHandler)
		api.DELETE("/devices", hb.SignOutOtherUserDevicesHandler)
		api.POST("/fcm", hb.UpdateFCMTokenHandler)
		api.PUT("/safety-preferences", hb.UpdateSafetyPreferences)
		api.PUT("/trusted-providers", hb.UpdateTrustedProviders)
	}
}

func RegisterProviderRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/providers")
	api.Use(middleware.DeviceDetailsMiddleware()) // Extract device details

	{
		api.POST("/register", hb.RegisterProviderHandler)
		api.POST("/login", hb.AuthenticateProviderHandler)
		api.POST("/reset-password", hb.ResetProviderPasswordHandler)
		api.GET("/legal", hb.ProviderLegalDocumentation)

		public := api.Group("")
		public.GET("/id/:id", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByIDHandler)
		public.GET("/email/:email", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByEmailHandler)

		protected := api.Group("")
		protected.Use(
			middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, false),
			// middleware.DeviceAuthMiddlewareProvider(hb.ProviderRepo),
		)
		{
			protected.PUT("/update/:id", hb.UpdateProviderHandler)
			protected.DELETE("/delete/:id", hb.DeleteProviderHandler)
			protected.PUT("/advance-verify/:id", hb.AdvanceVerifyProviderHandler)
			protected.DELETE("/revoke/:id", hb.RevokeProviderAuthTokenHandler)
			protected.PUT("/password/:id", hb.UpdateProviderPasswordHandler)
			protected.POST("/fcm", hb.UpdateProviderFCMTokenHandler)

			// Provider device endpoints
			protected.GET("/devices", hb.GetProviderDevicesHandler)
			protected.DELETE("/devices", hb.SignOutOtherProviderDevicesHandler)

			//Timeslot management endpoints
			protected.PUT("/timeslots/:id", hb.SetupTimeslotsHandler)
			protected.POST("/timeslots", hb.GetTimeslotsHandler)
			protected.DELETE("/timeslot", hb.DeleteTimeslotHandler)
			protected.GET("/booking/:bookingId", hb.VerifyBooking)
		}
	}
}

func RegisterAdminRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.DeviceDetailsMiddleware()) // Extract device details
	{
		adminGroup.Use(middleware.JWTAuthAdminMiddleware())
		adminGroup.GET("/users", hb.GetAllUsersHandler)
		adminGroup.GET("/providers", hb.GetAllProvidersHandler)
		adminGroup.POST("/legal", hb.AdminLegalDocumentation)
		adminGroup.GET("/health", hb.AdminHandler.SystemHealthHandler)
	}
}

func RegisterAIRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	aiGroup := r.Group("/api/ai")
	aiGroup.Use(
		middleware.DeviceDetailsMiddleware(),
		middleware.JWTAuthUserMiddleware(hb.UserRepo),
	)
	{
		aiGroup.POST("/stt", hb.AISTTHandler)
		aiGroup.POST("/chat", hb.AIChatHandler)
	}
}

func RegisterHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Hi, I'm Bloomify"})
	})
}

func RegisterBookingRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	bookingGroup := r.Group("/api/booking")
	bookingGroup.GET("/services/:region", hb.GetAvailableServices)
	bookingGroup.POST("/service", hb.GetServiceByID)

	bookingGroup.Use(
		middleware.DeviceDetailsMiddleware(),
		middleware.JWTAuthUserMiddleware(hb.UserRepo),
		// middleware.DeviceAuthMiddlewareUser(hb.UserRepo),
	)
	{
		bookingGroup.POST("/session", hb.InitiateSession)
		bookingGroup.PUT("/session/:sessionID", hb.UpdateSession)
		bookingGroup.POST("/confirm", hb.ConfirmBooking)
		bookingGroup.DELETE("/session/:sessionID", hb.CancelSession)
		bookingGroup.GET("/directions", hb.GetDirections)
		bookingGroup.GET("/geocode", hb.GeocodeAddress)
		bookingGroup.GET("/reverse", hb.ReverseGeocode)
		bookingGroup.POST("/payment", hb.GetPaymentIntent)
		bookingGroup.POST("/nearby", hb.MatchNearbyProviders)
	}
}

func RegisterStorageRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	public := r.Group("/storage")
	public.Use(middleware.DeviceDetailsMiddleware())
	{
		public.POST("/upload", hb.UploadFileHandler)
		public.GET("/public", hb.GetDownloadURLHandler)
		public.POST("/kyp/upload", hb.KYPUploadFileHandler)
	}

	// Protected routes for KYP downloads (admin-only)
	protected := r.Group("/storage")
	protected.Use(middleware.DeviceDetailsMiddleware(), middleware.JWTAuthAdminMiddleware())
	{
		// KYP file downloads
		protected.GET("/kyp", hb.KYPGetDownloadURLHandler)
	}
}

func RegisterRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Device-ID", "X-Device-Name"},
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
	RegisterStorageRoutes(r, hb)
}
