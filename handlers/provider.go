// File: bloomify/handlers/provider.go
package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProviderHandler encapsulates provider service and its handlers.
type ProviderHandler struct {
	Service provider.ProviderService
}

// NewProviderHandler returns a new ProviderHandler instance.
func NewProviderHandler(ps provider.ProviderService) *ProviderHandler {
	return &ProviderHandler{Service: ps}
}

// RegisterProviderHandler handles POST /providers.
func (h *ProviderHandler) RegisterProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var reqProvider models.Provider
	if err := c.ShouldBindJSON(&reqProvider); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	createdProvider, err := h.Service.RegisterProvider(reqProvider)
	if err != nil {
		logger.Error("Failed to register provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register provider"})
		return
	}

	c.JSON(http.StatusCreated, createdProvider)
}

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

// AuthenticateProviderHandler handles POST /providers/authenticate.
func (h *ProviderHandler) AuthenticateProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	prov, err := h.Service.AuthenticateProvider(req.Email, req.Password)
	if err != nil {
		logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, prov)
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

// RevokeProviderAuthTokenHandler handles DELETE /providers/revoke/:id.
func (h *ProviderHandler) RevokeProviderAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	providerID := c.Param("id")
	if err := h.Service.RevokeProviderAuthToken(providerID); err != nil {
		logger.Error("Failed to revoke provider auth token", zap.String("id", providerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke auth token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
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
