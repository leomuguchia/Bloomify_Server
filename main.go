// File: bloomify/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bloomify/config"
	"bloomify/database"
	providerRepo "bloomify/database/repository/provider"
	schedulerRepo "bloomify/database/repository/scheduler"
	userRepoPkg "bloomify/database/repository/user"
	"bloomify/handlers"
	"bloomify/middleware"
	"bloomify/routes"
	"bloomify/services/booking"
	"bloomify/services/provider"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration and initialize the logger.
	config.LoadConfig()
	logger := utils.GetLogger()

	// Initialize the database and caches.
	database.InitDB()
	utils.InitCache()
	utils.InitAuthCache()

	// Initialize Cloudinary Storage Service via the utils.Cloudinary() helper.
	cloudinaryStorageService, err := utils.Cloudinary()
	if err != nil {
		logger.Sugar().Fatalf("main: failed to initialize cloudinary storage service: %v", err)
	}

	// Create a new Gin router with desired middleware.
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(utils.ErrorHandler())
	router.Use(gin.Logger())
	router.Use(middleware.RateLimitMiddleware())
	router.Use(middleware.GeolocationMiddleware())

	// Setup repositories.
	provRepo := providerRepo.NewMongoProviderRepo()
	userRepo := userRepoPkg.NewMongoUserRepo()

	// Setup services.
	userService := &user.DefaultUserService{
		Repo: userRepo,
	}
	handlers.SetUserService(userService)

	providerService := &provider.DefaultProviderService{
		Repo: provRepo,
	}
	providerHandler := handlers.NewProviderHandler(providerService)

	matchingServiceInstance := &booking.DefaultMatchingService{
		ProviderRepo: provRepo,
	}

	schedulingEngineInstance := &booking.DefaultSchedulingEngine{
		Repo:           schedulerRepo.NewMongoSchedulerRepo(),
		PaymentHandler: nil,
	}
	bookingService := &booking.DefaultBookingSessionService{
		MatchingSvc:     matchingServiceInstance,
		SchedulerEngine: schedulingEngineInstance,
	}
	bookingHandler := handlers.NewBookingHandler(bookingService)
	handlers.SetBookingHandler(bookingHandler)

	// Create the admin handler with elevated privileges.
	adminHandler := handlers.NewAdminHandler(userService, providerService)

	// Create the storage handler using the Cloudinary storage service.
	storageHandler := handlers.NewStorageHandler(cloudinaryStorageService)

	// Assemble the handler bundle.
	handlerBundle := &handlers.HandlerBundle{
		ProviderRepo:                   provRepo,
		UserRepo:                       userRepo,
		GetProviderByIDHandler:         providerHandler.GetProviderByIDHandler,
		GetProviderByEmailHandler:      providerHandler.GetProviderByEmailHandler,
		RegisterProviderHandler:        providerHandler.RegisterProviderHandler,
		UpdateProviderHandler:          providerHandler.UpdateProviderHandler,
		DeleteProviderHandler:          providerHandler.DeleteProviderHandler,
		AuthenticateProviderHandler:    providerHandler.AuthenticateProviderHandler,
		KYPVerificationHandler:         handlers.KYPVerificationHandler,
		AdvanceVerifyProviderHandler:   providerHandler.AdvanceVerifyProviderHandler,
		RevokeProviderAuthTokenHandler: providerHandler.RevokeProviderAuthTokenHandler,
		SetupTimeslotsHandler:          providerHandler.SetupTimeslotsHandler,
		// Booking endpoints
		InitiateSession: bookingHandler.InitiateSession,
		UpdateSession:   bookingHandler.UpdateSession,
		ConfirmBooking:  bookingHandler.ConfirmBooking,
		CancelSession:   bookingHandler.CancelSession,
		// AI endpoints
		AIRecommendHandler: handlers.AIRecommendHandler,
		AISuggestHandler:   handlers.AISuggestHandler,
		AutoBookHandler:    handlers.AutoBookHandler,
		// User endpoints
		RegisterUserHandler:          handlers.RegisterUserHandler,
		AuthenticateUserHandler:      handlers.AuthenticateUserHandler,
		GetUserByIDHandler:           handlers.GetUserByIDHandler,
		GetUserByEmailHandler:        handlers.GetUserByEmailHandler,
		UpdateUserHandler:            handlers.UpdateUserHandler,
		DeleteUserHandler:            handlers.DeleteUserHandler,
		RevokeUserAuthTokenHandler:   handlers.RevokeUserAuthTokenHandler,
		UpdateUserPreferencesHandler: handlers.UpdateUserPreferencesHandler,
		UpdateUserPasswordHandler:    handlers.UpdateUserPasswordHandler,
		// Admin endpoints
		AdminHandler: adminHandler,
		// Storage endpoints
		StorageHandler: storageHandler,
		// OTP endpoint
		VerifyOTPHandler: handlers.VerifyOTPHandler,
	}

	// Register all application routes.
	routes.RegisterRoutes(router, handlerBundle)
	// Register storage-specific routes.
	routes.RegisterStorageRoutes(router, storageHandler)

	port := config.AppConfig.AppPort
	if port == "" {
		port = "8080"
	}

	// Bind the server to all network interfaces.
	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: router,
	}

	logger.Sugar().Infof("Starting server on %s...", srv.Addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Sugar().Fatalf("main: server failed to start: %v", err)
		}
	}()

	// Wait for an OS signal to gracefully shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Sugar().Info("main: server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Sugar().Fatalf("main: server forced to shutdown: %v", err)
	}

	logger.Sugar().Info("main: server stopped gracefully")
}
