package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/provider"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// providerService holds the ProviderService instance and should be set during app initialization.
var providerService provider.ProviderService

// SetProviderService sets the provider service instance.
func SetProviderService(ps provider.ProviderService) {
	providerService = ps
}

// RegisterProviderHandler handles POST /providers.
// It registers a new provider by validating input, hashing the password,
// and persisting the provider.
func RegisterProviderHandler(c *gin.Context) {
	logger := getLogger(c)

	var reqProvider models.Provider
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	createdProvider, err := providerService.RegisterProvider(reqProvider)
	if err != nil {
		logger.Error("Failed to register provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register provider"})
		return
	}

	c.JSON(http.StatusCreated, createdProvider)
}

// GetProviderByIDHandler handles GET /providers/:id.
// It retrieves a provider by its ID.
func GetProviderByIDHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")
	prov, err := providerService.GetProviderByID(id)
	if err != nil {
		logger.Error("Provider not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, prov)
}

// GetProviderByEmailHandler handles GET /providers/email/:email.
// It retrieves a provider by its email.
func GetProviderByEmailHandler(c *gin.Context) {
	logger := getLogger(c)
	email := c.Param("email")
	prov, err := providerService.GetProviderByEmail(email)
	if err != nil {
		logger.Error("Provider not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, prov)
}

// UpdateProviderHandler handles PUT /providers/:id.
// It updates an existing provider's allowed fields.
func UpdateProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")

	var reqProvider models.Provider
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Ensure the provider ID in the URL is used.
	reqProvider.ID = id

	updatedProvider, err := providerService.UpdateProvider(reqProvider)
	if err != nil {
		logger.Error("Failed to update provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}
	c.JSON(http.StatusOK, updatedProvider)
}

// DeleteProviderHandler handles DELETE /providers/:id.
// It deletes the provider with the given ID.
func DeleteProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")
	if err := providerService.DeleteProvider(id); err != nil {
		logger.Error("Failed to delete provider", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

// AuthenticateProviderHandler handles POST /providers/authenticate.
// It verifies the provider's email and password.
func AuthenticateProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	prov, err := providerService.AuthenticateProvider(req.Email, req.Password)
	if err != nil {
		logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Return the provider data (sensitive fields excluded by service layer).
	c.JSON(http.StatusOK, prov)
}
