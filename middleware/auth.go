package middleware

import (
	"net/http"
	"strings"

	"bloomify/services"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
)

// AuthMiddleware verifies that the request contains a valid JWT in the Authorization header.
// It expects the header to be in the format "Bearer <token>" and, if valid, sets the user ID in the context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		tokenStr := parts[1]

		// Parse token using our custom claims defined in the auth service.
		jwtSecret := []byte(utils.AppConfig.JWTSecret)
		token, err := jwt.ParseWithClaims(tokenStr, &services.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			logger.Error("Invalid token", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Extract user ID from claims and set in context.
		if claims, ok := token.Claims.(*services.CustomClaims); ok {
			c.Set("userID", claims.UserID)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}
	}
}
