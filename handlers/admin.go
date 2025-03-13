// File: bloomify/handlers/admin.go
package handlers

import (
	"net/http"

	"bloomify/services/provider"
	"bloomify/services/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler encapsulates admin-level operations that can access multiple services.
type AdminHandler struct {
	UserService     user.UserService
	ProviderService provider.ProviderService
	// Add other services as needed.
}

// NewAdminHandler creates a new instance of AdminHandler.
func NewAdminHandler(us user.UserService, ps provider.ProviderService) *AdminHandler {
	return &AdminHandler{
		UserService:     us,
		ProviderService: ps,
	}
}

// GetAllUsersHandler is an example admin endpoint that retrieves all users.
func (ah *AdminHandler) GetAllUsersHandler(c *gin.Context) {
	users, err := ah.UserService.GetAllUsers()
	if err != nil {
		zap.L().Error("Failed to fetch all users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}
