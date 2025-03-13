// File: middleware/jwt_auth_admin.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Fixed static admin token for demonstration purposes.
const adminStaticToken = "admin-secret-token-1234"

// JWTAuthAdminMiddleware validates the JWT token for admin routes using a fixed token.
func JWTAuthAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve token from header.
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
