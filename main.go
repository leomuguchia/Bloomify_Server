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
	// Load configuration and initialize logger.
	config.LoadConfig()
	logger := utils.GetLogger()

	// Initialize database and caches.
	database.InitDB()
	utils.InitCache()
	utils.InitAuthCache()

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

	// Create the admin handler (elevated service) by passing user and provider services.
	adminHandler := handlers.NewAdminHandler(userService, providerService)

	// Create the handler bundle.
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
		RegisterUserHandler:        handlers.RegisterUserHandler,
		AuthenticateUserHandler:    handlers.AuthenticateUserHandler,
		GetUserByIDHandler:         handlers.GetUserByIDHandler,
		GetUserByEmailHandler:      handlers.GetUserByEmailHandler,
		UpdateUserHandler:          handlers.UpdateUserHandler,
		DeleteUserHandler:          handlers.DeleteUserHandler,
		RevokeUserAuthTokenHandler: handlers.RevokeUserAuthTokenHandler,
		// Admin endpoints
		AdminHandler: adminHandler,
	}

	// Register all routes including admin routes.
	routes.RegisterRoutes(router, handlerBundle)

	port := config.AppConfig.AppPort
	if port == "" {
		port = "8080"
	}

	// Create an HTTP server using the Gin router.
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	logger.Sugar().Infof("Starting server on port %s...", port)

	// Start the server in a separate goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Sugar().Fatalf("Server failed to start: %v", err)
		}
	}()

	// Listen for OS interrupt signals to gracefully shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Sugar().Info("Server is shutting down...")

	// Create a context with a timeout for graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Sugar().Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Sugar().Info("Server stopped gracefully")
}
