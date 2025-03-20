// File: bloomify/handlers/providercrud.go
package handlers

import (
	"bloomify/models"
	"bloomify/services/provider"
	"bloomify/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetProviderByIDHandler handles GET /providers/:id.
func (h *ProviderHandler) GetProviderByIDHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	prov, err := h.Service.GetProviderByID(c, id)
	if err != nil {
		logger.Error("Provider not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, prov)
}

// GetProviderByEmailHandler handles GET /providers/email/:email.
func (h *ProviderHandler) GetProviderByEmailHandler(c *gin.Context) {
	logger := utils.GetLogger()
	email := c.Param("email")
	prov, err := h.Service.GetProviderByEmail(c, email)
	if err != nil {
		logger.Error("Provider not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	c.JSON(http.StatusOK, prov)
}

// DeleteProviderHandler handles DELETE /providers/:id.
func (h *ProviderHandler) DeleteProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := h.Service.DeleteProvider(id); err != nil {
		logger.Error("Failed to delete provider", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

// AdvanceVerifyProviderHandler handles PUT /providers/advance-verify/:id.
func (h *ProviderHandler) AdvanceVerifyProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()
	providerID := c.Param("id")

	var advReq provider.AdvanceVerifyRequest
	if err := c.ShouldBindJSON(&advReq); err != nil {
		logger.Error("Invalid advanced verification request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedProvider, err := h.Service.AdvanceVerifyProvider(c, providerID, advReq)
	if err != nil {
		logger.Error("Failed to advanced verify provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to advanced verify provider"})
		return
	}

	c.JSON(http.StatusOK, updatedProvider)
}

func (h *ProviderHandler) UpdateProviderHandler(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.BindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Remove the id field if provided in the payload.
	delete(updates, "id")

	updatedProvider, err := h.Service.UpdateProvider(c, id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Provider updated successfully",
		"data":    updatedProvider,
	})
}

func (h *ProviderHandler) SetupTimeslotsHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Retrieve provider ID from the context (set by JWTAuthProviderMiddleware).
	providerIDValue, exists := c.Get("providerID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
		return
	}
	providerID, ok := providerIDValue.(string)
	if !ok || providerID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider ID in context"})
		return
	}

	// Bind the incoming JSON payload to SetupTimeslotsRequest.
	var req models.SetupTimeslotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid timeslot setup request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	dto, err := h.Service.SetupTimeslots(c, providerID, req)
	if err != nil {
		logger.Error("Failed to set up timeslots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set up timeslots", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Timeslot setup successful; provider status updated to active",
		"provider": dto,
	})
}
