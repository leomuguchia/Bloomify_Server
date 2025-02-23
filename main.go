package main

import (
	"log"
	"os"
	"time"

	"bloomify/database"
	"bloomify/routes"
	"bloomify/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func initConfig() {
	// Load .env file if it exists.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using environment variables")
	}
	// Initialize application configuration using our utils (e.g., via Viper).
	utils.InitConfig()
}

func initLogger() *zap.Logger {
	logger, err := zap.NewProduction() // Use zap.NewDevelopment() during development
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	// Make sure to flush any buffered log entries before the application exits.
	defer logger.Sync()
	return logger
}

func main() {
	// Initialize configuration and logger.
	initConfig()
	logger := initLogger()
	sugar := logger.Sugar()

	// Initialize database connection and run migrations.
	database.InitDB()

	// Create Gin router with advanced middleware.
	router := gin.New()
	router.Use(gin.Recovery())       // Recover from panics and return 500
	router.Use(utils.ErrorHandler()) // Custom error handler middleware from utils
	router.Use(gin.Logger())         // Default Gin logger

	// Setup CORS for cross-origin requests.
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Adjust this for production security
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register routes from separate modules.
	routes.RegisterAuthRoutes(router)
	routes.RegisterUserRoutes(router)
	routes.RegisterProviderRoutes(router)
	routes.RegisterBlockedRoutes(router)
	routes.RegisterBookingRoutes(router)
	routes.RegisterMatchingRoutes(router)
	routes.RegisterAIRoutes(router)
	routes.RegisterHealthRoute(router)

	// Start server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	sugar.Infof("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		sugar.Fatalf("Server failed to start: %v", err)
	}
}
