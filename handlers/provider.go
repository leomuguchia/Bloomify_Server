package handlers

import (
	"net/http"

	"bloomify/database/repository"
	"bloomify/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetProvidersHandler returns a list of providers.
func GetProvidersHandler(c *gin.Context) {
	logger := getLogger(c)
	providerRepo := repository.GormProviderRepo{}

	providers, err := providerRepo.GetAll()
	if err != nil {
		logger.Error("Failed to retrieve providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get providers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// GetProviderHandler returns details for a specific provider.
func GetProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")
	providerRepo := repository.GormProviderRepo{}

	provider, err := providerRepo.GetByID(id)
	if err != nil {
		logger.Error("Provider not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, provider)
}

// CreateProviderHandler creates a new provider.
func CreateProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		logger.Error("Invalid provider creation request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	providerRepo := repository.GormProviderRepo{}
	if err := providerRepo.Create(&provider); err != nil {
		logger.Error("Failed to create provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}
	c.JSON(http.StatusCreated, provider)
}

// UpdateProviderHandler updates provider information.
func UpdateProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")
	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	provider.ID = id // Ensure the ID is set.
	providerRepo := repository.GormProviderRepo{}
	if err := providerRepo.Update(&provider); err != nil {
		logger.Error("Failed to update provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}
	c.JSON(http.StatusOK, provider)
}

// DeleteProviderHandler deletes a provider.
func DeleteProviderHandler(c *gin.Context) {
	logger := getLogger(c)
	id := c.Param("id")
	providerRepo := repository.GormProviderRepo{}
	if err := providerRepo.Delete(id); err != nil {
		logger.Error("Failed to delete provider", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}
