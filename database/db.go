package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/spf13/viper"

	"bloomify/models"
)

// DB is the global database connection instance.
var DB *gorm.DB

// InitDB initializes the database connection and performs migrations.
func InitDB() {
	// Load database file from config; default to "backend.db".
	dbFile := viper.GetString("DB_FILE")
	if dbFile == "" {
		dbFile = "backend.db"
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established.")

	// Run migrations.
	runMigrations()
}

// runMigrations migrates the schema.
func runMigrations() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Provider{},
		&models.Blocked{},
		&models.Booking{},
		// &models.ServiceType{}, // Optional
	)
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
	log.Println("Database migrated successfully.")
}
