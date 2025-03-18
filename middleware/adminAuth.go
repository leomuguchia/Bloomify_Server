// File: middleware/jwt_auth_admin.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const adminStaticToken = "admin-secret-token-1234"

func JWTAuthAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate against fixed static admin token.
		if tokenString != adminStaticToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized admin access"})
			return
		}

		// Optionally, set an admin flag in context.
		c.Set("isAdmin", true)
		c.Next()
	}
}
