// File: bloomify/handlers/admin.go
package handlers

import (
	"net/http"

	"bloomify/services/admin"
	"bloomify/services/provider"
	"bloomify/services/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler encapsulates elevated admin-level operations.
type AdminHandler struct {
	UserService     user.UserService
	ProviderService provider.ProviderService
	AdminService    admin.AdminService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(us user.UserService, ps provider.ProviderService, as admin.AdminService) *AdminHandler {
	return &AdminHandler{
		UserService:     us,
		ProviderService: ps,
		AdminService:    as,
	}
}

// GetAllUsersHandler returns all users (with sensitive fields excluded).
func (ah *AdminHandler) GetAllUsersHandler(c *gin.Context) {
	users, err := ah.UserService.GetAllUsers()
	if err != nil {
		zap.L().Error("Failed to fetch all users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetAllProvidersHandler returns all providers (with sensitive fields excluded).
func (ah *AdminHandler) GetAllProvidersHandler(c *gin.Context) {
	providers, err := ah.ProviderService.GetAllProviders()
	if err != nil {
		zap.L().Error("Failed to fetch all providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
		return
	}
	c.JSON(http.StatusOK, providers)
}

func (ah *AdminHandler) AdminLegalDocumentation(c *gin.Context) {
	sections := ah.AdminService.GetLegalSectionsFor("both")

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"version": "v1.0",
		"data":    sections,
	})
}
