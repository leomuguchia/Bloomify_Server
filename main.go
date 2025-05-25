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
	recordsRepo "bloomify/database/repository/records"
	schedulerRepo "bloomify/database/repository/scheduler"
	timeslotRepo "bloomify/database/repository/timeslot"
	userRepoPkg "bloomify/database/repository/user"
	"bloomify/handlers"
	"bloomify/middleware"
	"bloomify/routes"
	"bloomify/services/booking"
	ai "bloomify/services/intelligence"
	"bloomify/services/notification"
	"bloomify/services/provider"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
)

func main() {
	config.LoadConfig()
	logger := utils.GetLogger()

	database.InitDB()
	utils.InitRedis()
	utils.FirebaseInit()

	cloudinaryStorageService, err := utils.Cloudinary()
	if err != nil {
		logger.Sugar().Fatalf("main: failed to initialize cloudinary storage service: %v", err)
	}

	// Create the Gin router.
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(utils.ErrorHandler())
	router.Use(gin.Logger())
	router.Use(middleware.RateLimitMiddleware())
	router.Use(middleware.GeolocationMiddleware())
	stripe.Key = config.AppConfig.StripeKey

	// repositories.
	provRepo := providerRepo.NewMongoProviderRepo()
	userRepo := userRepoPkg.NewMongoUserRepo()
	timeslotRepo := timeslotRepo.NewMongoTimeSlotRepo()
	schedulerRepo := schedulerRepo.NewMongoSchedulerRepo(timeslotRepo)
	recordsRepo := recordsRepo.NewMongoRecordRepo()

	// services.
	userService := &user.DefaultUserService{
		Repo: userRepo,
	}
	handlers.SetUserService(userService)

	providerService := &provider.DefaultProviderService{
		Repo:        provRepo,
		Timeslot:    timeslotRepo,
		RecordsRepo: recordsRepo,
	}
	providerHandler := handlers.NewProviderHandler(providerService)
	ProviderDeviceHandler := handlers.NewProviderDeviceHandler(providerService)
	UserDeviceHandler := handlers.NewUserDeviceHandler(userService)

	matchingServiceInstance := &booking.DefaultMatchingService{
		ProviderRepo: provRepo,
	}

	notificationService := &notification.DefaultNotificationService{
		User:     userService,
		Provider: providerService,
	}

	paymentHandler := booking.NewPaymentHandler(logger, userService)
	schedulingEngineInstance := &booking.DefaultSchedulingEngine{
		Repo:           schedulerRepo,
		PaymentHandler: paymentHandler,
		ProviderRepo:   provRepo,
		TimeslotsRepo:  timeslotRepo,
		UserService:    userService,
		Notification:   notificationService,
	}

	bookingService := &booking.DefaultBookingSessionService{
		MatchingSvc:     matchingServiceInstance,
		SchedulerEngine: schedulingEngineInstance,
		NotificationSvc: notificationService,
	}

	ctxStore := ai.NewRedisContextStore(utils.GetAIContextCacheClient(), 30*time.Minute)
	aiSvc := ai.NewDefaultAIService(
		config.AppConfig.GeminiAPIKey,
		ctxStore,
		bookingService,
	)

	bookingHandler := handlers.NewBookingHandler(bookingService, matchingServiceInstance, logger)
	adminHandler := handlers.NewAdminHandler(userService, providerService)
	storageHandler := handlers.NewStorageHandler(cloudinaryStorageService)
	aiHandler := handlers.NewDefaultAIHandler(aiSvc)

	// Assemble the handler bundle.
	handlerBundle := &handlers.HandlerBundle{
		// Provider endpoints.
		ProviderRepo:                   provRepo,
		UserRepo:                       userRepo,
		GetProviderByIDHandler:         providerHandler.GetProviderByIDHandler,
		GetProviderByEmailHandler:      providerHandler.GetProviderByEmailHandler,
		RegisterProviderHandler:        providerHandler.RegisterProviderHandler,
		UpdateProviderHandler:          providerHandler.UpdateProviderHandler,
		DeleteProviderHandler:          providerHandler.DeleteProviderHandler,
		AuthenticateProviderHandler:    providerHandler.AuthenticateProviderHandler,
		AdvanceVerifyProviderHandler:   providerHandler.AdvanceVerifyProviderHandler,
		RevokeProviderAuthTokenHandler: providerHandler.RevokeProviderAuthTokenHandler,
		UpdateProviderPasswordHandler:  providerHandler.UpdateProviderPasswordHandler,

		//provider timeslots management
		SetupTimeslotsHandler: providerHandler.SetupTimeslotsHandler,
		GetTimeslotsHandler:   providerHandler.GetTimeslotsHandler,
		DeleteTimeslotHandler: providerHandler.DeleteTimeslotHandler,

		// Provider device endpoints.
		GetProviderDevicesHandler:          ProviderDeviceHandler.GetProviderDevicesHandler,
		SignOutOtherProviderDevicesHandler: ProviderDeviceHandler.SignOutOtherProviderDevicesHandler,

		// Booking endpoints.
		InitiateSession:      bookingHandler.InitiateSession,
		UpdateSession:        bookingHandler.UpdateSession,
		ConfirmBooking:       bookingHandler.ConfirmBooking,
		CancelSession:        bookingHandler.CancelSession,
		GetAvailableServices: bookingHandler.GetAvailableServices,
		GetDirections:        bookingHandler.GetDirections,
		GetPaymentIntent:     bookingHandler.GetPaymentIntent,
		MatchNearbyProviders: bookingHandler.MatchNearbyProviders,

		// AI endpoints.
		AISTTHandler:  aiHandler.AISTTHandler,
		AIChatHandler: aiHandler.HandleAIRequest,

		// User endpoints.
		RegisterUserHandler:        handlers.RegisterUserHandler,
		AuthenticateUserHandler:    handlers.AuthenticateUserHandler,
		GetUserByIDHandler:         handlers.GetUserByIDHandler,
		GetUserByEmailHandler:      handlers.GetUserByEmailHandler,
		UpdateUserHandler:          handlers.UpdateUserHandler,
		DeleteUserHandler:          handlers.DeleteUserHandler,
		RevokeUserAuthTokenHandler: handlers.RevokeUserAuthTokenHandler,
		UpdateUserPasswordHandler:  handlers.UpdateUserPasswordHandler,

		// User device endpoints.
		GetUserDevicesHandler:          UserDeviceHandler.GetUserDevicesHandler,
		SignOutOtherUserDevicesHandler: UserDeviceHandler.SignOutOtherUserDevicesHandler,
		UpdateFCMTokenHandler:          UserDeviceHandler.UpdateFCMTokenHandler,

		// OTP endpoints.
		VerifyOTPHandler: handlers.VerifyOTPHandler,

		// Password reset endpoints for users.
		ResetPasswordHandler: handlers.ResetUserPasswordHandler,

		// Provider forgot password endpoint.
		ResetProviderPasswordHandler: handlers.ResetProviderPasswordHandler,

		// Admin endpoints.
		AdminHandler: adminHandler,

		// Storage endpoints.
		StorageHandler:           storageHandler,
		UploadFileHandler:        storageHandler.UploadFileHandler,
		GetDownloadURLHandler:    storageHandler.GetDownloadURLHandler,
		KYPUploadFileHandler:     storageHandler.KYPUploadFileHandler,
		KYPGetDownloadURLHandler: storageHandler.KYPGetDownloadURLHandler,
	}

	// Register routes with the assembled handler bundle.
	routes.RegisterRoutes(router, handlerBundle)

	// Start the HTTP server.
	port := config.AppConfig.AppPort
	if port == "" {
		port = "8080"
	}
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
