package utils

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global logger instance
var Logger *zap.Logger

// InitializeLogger sets up the logging configuration
func InitializeLogger() {
	var cfg zap.Config

	if IsProduction() {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Create logger
	var err error
	Logger, err = cfg.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

// GetLogger retrieves the global logger
func GetLogger() *zap.Logger {
	if Logger == nil {
		InitializeLogger()
	}
	return Logger
}
