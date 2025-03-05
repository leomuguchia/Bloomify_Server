// middleware/jwt_auth.go
package middleware

import (
	providerRepo "bloomify/database/repository/provider"
	"bloomify/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate the token signature and expiration.
		token, err := utils.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Compute the token hash.
		computedHash := utils.HashToken(tokenString)

		// Query the database using the token hash.
		prov, err := providerRepo.GetByTokenHash(computedHash)
		if err != nil || prov == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token mismatch or provider not found"})
			return
		}

		// Set provider information in context if needed.
		c.Set("providerID", prov.ID)
		c.Next()
	}
}
