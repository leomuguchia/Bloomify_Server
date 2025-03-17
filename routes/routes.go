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
		api.Use(
			middleware.JWTAuthUserMiddleware(hb.UserRepo),
			middleware.DeviceAuthMiddlewareUser(hb.UserRepo),
		)

		api.PUT("/preferences/:id", hb.UpdateUserPreferencesHandler)
		api.GET("/id/:id", hb.GetUserByIDHandler)
		api.GET("/email/:email", hb.GetUserByEmailHandler)
		api.PUT("/update/:id", hb.UpdateUserHandler)
		api.DELETE("/delete/:id", hb.DeleteUserHandler)
		api.DELETE("/revoke/:id", hb.RevokeUserAuthTokenHandler)
		api.PUT("/password/:id", hb.UpdateUserPasswordHandler)
		api.GET("/devices", hb.GetDevicesHandler)
		api.DELETE("/devices", hb.SignOutOtherDevicesHandler)
	}
}

func RegisterProviderRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/providers")
	api.Use(middleware.DeviceDetailsMiddleware()) // Extract device details for all routes

	{
		api.POST("/register", hb.RegisterProviderHandler)
		api.POST("/login", hb.AuthenticateProviderHandler)
		api.POST("/kyp/verify", hb.KYPVerificationHandler)

		// Public routes that still require JWT verification but no device authentication
		public := api.Group("")
		public.GET("/id/:id", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByIDHandler)
		public.GET("/email/:email", middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, true), hb.GetProviderByEmailHandler)

		// Protected routes - require both JWT and Device authentication
		protected := api.Group("")
		protected.Use(
			middleware.JWTAuthProviderMiddleware(hb.ProviderRepo, false),
			middleware.DeviceAuthMiddlewareProvider(hb.ProviderRepo),
		)

		protected.PATCH("/update/:id", hb.UpdateProviderHandler)
		protected.DELETE("/delete/:id", hb.DeleteProviderHandler)
		protected.PUT("/advance-verify/:id", hb.AdvanceVerifyProviderHandler)
		protected.DELETE("/revoke/:id", hb.RevokeProviderAuthTokenHandler)
		protected.PUT("/create-timeslots/:id", hb.SetupTimeslotsHandler)
	}
}

func RegisterAIRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	api := r.Group("/api/ai")
	{
		api.Use(middleware.JWTAuthUserMiddleware(hb.UserRepo))
		api.POST("/recommend", hb.AIRecommendHandler)
		api.POST("/suggest", hb.AISuggestHandler)
		api.POST("/auto-book", hb.AutoBookHandler)
	}
}

func RegisterHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Hi, I'm Bloomify"})
	})
}

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

func RegisterAdminRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	adminGroup := r.Group("/api/admin")
	{
		adminGroup.Use(middleware.JWTAuthAdminMiddleware())
		adminGroup.GET("/users", hb.AdminHandler.GetAllUsersHandler)
		adminGroup.GET("/providers", hb.AdminHandler.GetAllProvidersHandler)
	}
}

func RegisterStorageRoutes(router *gin.Engine, storageHandler *handlers.StorageHandler) {
	storageGroup := router.Group("/storage")
	{
		storageGroup.POST("/:type/:bucket/upload", storageHandler.UploadFileHandler)
		storageGroup.GET("/:type/:bucket/:filename", storageHandler.GetDownloadURLHandler)
	}
}

func RegisterOTPRoutes(r *gin.Engine, hb *handlers.HandlerBundle) {
	r.POST("/api/verify-otp", middleware.DeviceDetailsMiddleware(), hb.VerifyOTPHandler)
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
	RegisterOTPRoutes(r, hb)
}
