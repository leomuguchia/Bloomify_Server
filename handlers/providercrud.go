// File: bloomify/handlers/providercrud.go
package handlers

import (
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
	// Check if the context has full access flag set.
	fullAccess := false
	if val, exists := c.Get("isProviderFullAccess"); exists {
		if fa, ok := val.(bool); ok {
			fullAccess = fa
		}
	}
	prov, err := h.Service.GetProviderByID(c, id, fullAccess)
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
	// Check if the context has full access flag set.
	fullAccess := false
	if val, exists := c.Get("isProviderFullAccess"); exists {
		if fa, ok := val.(bool); ok {
			fullAccess = fa
		}
	}
	prov, err := h.Service.GetProviderByEmail(c.Request.Context(), email, fullAccess)
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
	fullAccess := false
	// Check if the context has full access flag set.
	if val, exists := c.Get("isProviderFullAccess"); exists {
		if fa, ok := val.(bool); ok {
			fullAccess = fa
		}
	}

	var advReq provider.AdvanceVerifyRequest
	if err := c.ShouldBindJSON(&advReq); err != nil {
		logger.Error("Invalid advanced verification request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	updatedProvider, err := h.Service.AdvanceVerifyProvider(c.Request.Context(), providerID, advReq, fullAccess)
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

	updatedProvider, err := h.Service.UpdateProvider(c.Request.Context(), id, updates)
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

// UpdateProviderPasswordHandler handles PUT /providers/password/:id.
// It expects a JSON payload with "currentPassword" and "newPassword".
func (h *ProviderHandler) UpdateProviderPasswordHandler(c *gin.Context) {
	logger := utils.GetLogger()
	providerID := c.Param("id")

	// Extract device details from context (set by DeviceDetailsMiddleware).
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID"})
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid update password request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedProvider, err := providerService.UpdateProviderPassword(providerID, req.CurrentPassword, req.NewPassword, deviceID.(string))
	if err != nil {
		logger.Error("Failed to update provider password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedProvider)
}
