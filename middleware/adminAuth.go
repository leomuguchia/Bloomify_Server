package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const adminStaticToken = "MUGUCHIA_aDMIN"

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

		c.Set("adminToken", tokenString)
		c.Set("isAdmin", true)
		c.Next()
	}
}
