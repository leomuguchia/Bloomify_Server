package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorResponse defines the structure of error responses
type ErrorResponse struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// HandleErrors is a middleware to catch panics and return structured errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				Logger := GetLogger()
				Logger.Error("Unhandled panic", zap.Any("error", err))

				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Message: "Internal Server Error",
					Details: "An unexpected error occurred. Please try again later.",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// JSONError sends a standardized JSON error response
func JSONError(c *gin.Context, status int, message string, details string) {
	Logger := GetLogger()
	Logger.Warn(message, zap.String("details", details))
	c.JSON(status, ErrorResponse{Message: message, Details: details})
}
