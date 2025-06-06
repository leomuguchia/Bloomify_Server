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
	"bloomify/services/admin"
	"bloomify/services/booking"
	ai "bloomify/services/intelligence"
	"bloomify/services/notification"
	"bloomify/services/provider"
	"bloomify/services/storage"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
)

func main() {
	// Initialization
	config.LoadConfig()
	logger := utils.GetLogger()
	database.InitDB()
	utils.InitRedis()
	utils.FirebaseInit()

	// Gin setup
	router := gin.New()
	router.Use(gin.Recovery(), utils.ErrorHandler(), gin.Logger())
	router.Use(middleware.RateLimitMiddleware(), middleware.GeolocationMiddleware())
	stripe.Key = config.AppConfig.StripeKey

	// repos
	provRepo := providerRepo.NewMongoProviderRepo()
	userRepo := userRepoPkg.NewMongoUserRepo()
	timeslotRepo := timeslotRepo.NewMongoTimeSlotRepo()
	schedulerRepo := schedulerRepo.NewMongoSchedulerRepo(timeslotRepo)
	recordsRepo := recordsRepo.NewMongoRecordRepo()

	// services
	userService := &user.DefaultUserService{Repo: userRepo}
	adminService := &admin.DefaultAdminService{}

	providerService := &provider.DefaultProviderService{
		Repo:        provRepo,
		Timeslot:    timeslotRepo,
		RecordsRepo: recordsRepo,
	}

	notificationService := &notification.DefaultNotificationService{
		User:     userService,
		Provider: providerService,
	}

	matchingService := &booking.DefaultMatchingService{ProviderRepo: provRepo}

	paymentHandler := booking.NewPaymentHandler(logger, userService)

	schedulingEngine := &booking.DefaultSchedulingEngine{
		Repo:           schedulerRepo,
		PaymentHandler: paymentHandler,
		ProviderRepo:   provRepo,
		TimeslotsRepo:  timeslotRepo,
		UserService:    userService,
		Notification:   notificationService,
	}

	bookingService := &booking.DefaultBookingSessionService{
		MatchingSvc:     matchingService,
		SchedulerEngine: schedulingEngine,
		NotificationSvc: notificationService,
	}

	storageService, err := storage.NewFirebaseStorageService(
		config.FirebaseServiceAccountKeyPath,
		utils.BucketName,
	)
	if err != nil {
		logger.Sugar().Fatalf("failed to initialize storage service: %v", err)
	}

	aiCtxStore := ai.NewRedisContextStore(utils.GetAIContextCacheClient(), 30*time.Minute)
	aiService := ai.NewDefaultAIService(
		config.AppConfig.GeminiAPIKey,
		aiCtxStore,
		bookingService,
	)

	// handlers
	providerHandler := handlers.NewProviderHandler(providerService, adminService)
	providerDeviceHandler := handlers.NewProviderDeviceHandler(providerService)
	userHandler := handlers.NewUserHandler(userService, providerService, adminService)
	bookingHandler := handlers.NewBookingHandler(bookingService, matchingService, logger)
	adminHandler := handlers.NewAdminHandler(userService, providerService, adminService)
	storageHandler := handlers.NewStorageHandler(storageService)
	aiHandler := handlers.NewDefaultAIHandler(aiService)

	// handlerbundle assembly
	handlerBundle := &handlers.HandlerBundle{
		// Provider endpoints
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
		SetupTimeslotsHandler:          providerHandler.SetupTimeslotsHandler,
		GetTimeslotsHandler:            providerHandler.GetTimeslotsHandler,
		DeleteTimeslotHandler:          providerHandler.DeleteTimeslotHandler,
		ResetProviderPasswordHandler:   providerHandler.ResetProviderPasswordHandler,

		// Provider device endpoints
		GetProviderDevicesHandler:          providerDeviceHandler.GetProviderDevicesHandler,
		SignOutOtherProviderDevicesHandler: providerDeviceHandler.SignOutOtherProviderDevicesHandler,
		ProviderLegalDocumentation:         providerHandler.ProviderLegalDocumentation,

		// Booking endpoints
		InitiateSession:      bookingHandler.InitiateSession,
		UpdateSession:        bookingHandler.UpdateSession,
		ConfirmBooking:       bookingHandler.ConfirmBooking,
		CancelSession:        bookingHandler.CancelSession,
		GetAvailableServices: bookingHandler.GetAvailableServices,
		GetServiceByID:       bookingHandler.GetServiceByID,
		GetDirections:        bookingHandler.GetDirections,
		GetPaymentIntent:     bookingHandler.GetPaymentIntent,
		MatchNearbyProviders: bookingHandler.MatchNearbyProviders,

		// AI endpoints
		AISTTHandler:  aiHandler.AISTTHandler,
		AIChatHandler: aiHandler.HandleAIRequest,

		// User endpoints
		RegisterUserHandler:            userHandler.RegisterUserHandler,
		AuthenticateUserHandler:        userHandler.AuthenticateUserHandler,
		GetUserByIDHandler:             userHandler.GetUserByIDHandler,
		GetUserByEmailHandler:          userHandler.GetUserByEmailHandler,
		UpdateUserHandler:              userHandler.UpdateUserHandler,
		DeleteUserHandler:              userHandler.DeleteUserHandler,
		RevokeUserAuthTokenHandler:     userHandler.RevokeUserAuthTokenHandler,
		UpdateUserPasswordHandler:      userHandler.UpdateUserPasswordHandler,
		UserLegalDocumentation:         userHandler.UserLegalDocumentation,
		GetUserDevicesHandler:          userHandler.GetUserDevicesHandler,
		SignOutOtherUserDevicesHandler: userHandler.SignOutOtherUserDevicesHandler,
		UpdateFCMTokenHandler:          userHandler.UpdateFCMTokenHandler,
		ResetPasswordHandler:           userHandler.ResetUserPasswordHandler,
		UpdateSafetyPreferences:        userHandler.UpdateSafetyPreferences,
		UpdateTrustedProviders:         userHandler.UpdateTrustedProviders,

		// Admin endpoints
		AdminHandler:            adminHandler,
		AdminLegalDocumentation: adminHandler.AdminLegalDocumentation,
		GetAllProvidersHandler:  adminHandler.GetAllProvidersHandler,
		GetAllUsersHandler:      adminHandler.GetAllUsersHandler,

		// Storage endpoints
		StorageHandler:           storageHandler,
		UploadFileHandler:        storageHandler.UploadFileHandler,
		GetDownloadURLHandler:    storageHandler.GetDownloadURLHandler,
		KYPUploadFileHandler:     storageHandler.KYPUploadFileHandler,
		KYPGetDownloadURLHandler: storageHandler.KYPGetDownloadURLHandler,
	}

	// Register all routes
	routes.RegisterRoutes(router, handlerBundle)

	// Start HTTP server
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

	// Graceful shutdown
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
