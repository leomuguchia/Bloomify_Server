// File: bloomify/main.go
package main

import (
	"bloomify/database"
	providerRepo "bloomify/database/repository/provider"
	userRepoPkg "bloomify/database/repository/user"
	"bloomify/handlers"
	"bloomify/middleware"
	"bloomify/routes"
	"bloomify/services/user"
	"bloomify/utils"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize config, logger, database, etc.
	utils.LoadConfig()
	logger := utils.GetLogger()
	database.InitDB()

	// Create the Gin router.
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(utils.ErrorHandler())
	router.Use(gin.Logger())
	// Global middlewares: rate limiting and geolocation.
	router.Use(middleware.RateLimitMiddleware())
	router.Use(middleware.GeolocationMiddleware())

	// Instantiate dependencies.
	provRepo := providerRepo.NewMongoProviderRepo()
	// Instantiate user repository.
	userRepo := userRepoPkg.NewMongoUserRepo()
	// Instantiate the user service with its repository.
	userService := &user.DefaultUserService{
		Repo: userRepo,
	}
	// Set the user service in the handlers package so the user handlers can access it.
	handlers.SetUserService(userService)

	// Construct the handler bundle with both provider and user endpoints.
	handlerBundle := &handlers.HandlerBundle{
		// Provider endpoints
		GetProviderByIDHandler:      handlers.GetProviderByIDHandler,
		GetProviderByEmailHandler:   handlers.GetProviderByEmailHandler,
		RegisterProviderHandler:     handlers.RegisterProviderHandler,
		UpdateProviderHandler:       handlers.UpdateProviderHandler,
		DeleteProviderHandler:       handlers.DeleteProviderHandler,
		AuthenticateProviderHandler: handlers.AuthenticateProviderHandler,
		KYPVerificationHandler:      handlers.KYPVerificationHandler,

		// Booking endpoints
		InitiateSession: handlers.InitiateSession,
		UpdateSession:   handlers.UpdateSession,
		ConfirmBooking:  handlers.ConfirmBooking,

		// AI endpoints
		AIRecommendHandler: handlers.AIRecommendHandler,
		AISuggestHandler:   handlers.AISuggestHandler,
		AutoBookHandler:    handlers.AutoBookHandler,

		// User endpoints
		RegisterUserHandler:     handlers.RegisterUserHandler,
		AuthenticateUserHandler: handlers.AuthenticateUserHandler,
		GetUserByIDHandler:      handlers.GetUserByIDHandler,
		GetUserByEmailHandler:   handlers.GetUserByEmailHandler,
		UpdateUserHandler:       handlers.UpdateUserHandler,
		DeleteUserHandler:       handlers.DeleteUserHandler,
	}

	// Register all routes with dependencies.
	routes.RegisterRoutes(router, provRepo, handlerBundle)

	// Start the server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Sugar().Infof("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		logger.Sugar().Fatalf("Server failed to start: %v", err)
	}
}
