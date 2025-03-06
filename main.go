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
	utils.LoadConfig()
	logger := utils.GetLogger()
	database.InitDB()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(utils.ErrorHandler())
	router.Use(gin.Logger())
	router.Use(middleware.RateLimitMiddleware())
	router.Use(middleware.GeolocationMiddleware())

	provRepo := providerRepo.NewMongoProviderRepo()
	userRepo := userRepoPkg.NewMongoUserRepo()
	userService := &user.DefaultUserService{
		Repo: userRepo,
	}
	handlers.SetUserService(userService)

	handlerBundle := &handlers.HandlerBundle{
		GetProviderByIDHandler:      handlers.GetProviderByIDHandler,
		GetProviderByEmailHandler:   handlers.GetProviderByEmailHandler,
		RegisterProviderHandler:     handlers.RegisterProviderHandler,
		UpdateProviderHandler:       handlers.UpdateProviderHandler,
		DeleteProviderHandler:       handlers.DeleteProviderHandler,
		AuthenticateProviderHandler: handlers.AuthenticateProviderHandler,
		KYPVerificationHandler:      handlers.KYPVerificationHandler,
		InitiateSession:             handlers.InitiateSession,
		UpdateSession:               handlers.UpdateSession,
		ConfirmBooking:              handlers.ConfirmBooking,
		AIRecommendHandler:          handlers.AIRecommendHandler,
		AISuggestHandler:            handlers.AISuggestHandler,
		AutoBookHandler:             handlers.AutoBookHandler,
		RegisterUserHandler:         handlers.RegisterUserHandler,
		AuthenticateUserHandler:     handlers.AuthenticateUserHandler,
		GetUserByIDHandler:          handlers.GetUserByIDHandler,
		GetUserByEmailHandler:       handlers.GetUserByEmailHandler,
		UpdateUserHandler:           handlers.UpdateUserHandler,
		DeleteUserHandler:           handlers.DeleteUserHandler,
	}

	routes.RegisterRoutes(router, provRepo, handlerBundle)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Sugar().Infof("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		logger.Sugar().Fatalf("Server failed to start: %v", err)
	}
}
