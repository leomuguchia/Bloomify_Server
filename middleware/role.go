package middleware

import (
	providerRepo "bloomify/database/repository/provider"
	userRepo "bloomify/database/repository/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RoleBasedAuthMiddleware(userRepo userRepo.UserRepository, providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetHeader("role")

		switch role {
		case "user":
			JWTAuthUserMiddleware(userRepo)(c)
		case "provider":
			JWTAuthProviderMiddleware(providerRepo, false)(c) // false = not optional
		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Invalid or missing 'role' header. Expected 'user' or 'provider'.",
			})
			return
		}
	}
}
