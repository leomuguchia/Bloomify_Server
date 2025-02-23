package main

import (
	"bloomify/database"
	"bloomify/routes"
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

	// Register all routes in one centralized function.
	routes.RegisterRoutes(router)

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
