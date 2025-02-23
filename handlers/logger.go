package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// getLogger retrieves a Zap logger from the Gin context or creates a new one.
func getLogger(c *gin.Context) *zap.Logger {
	if l, exists := c.Get("logger"); exists {
		if logger, ok := l.(*zap.Logger); ok {
			return logger
		}
	}
	logger, _ := zap.NewProduction()
	return logger
}
