package handlers

import (
	"bloomify/models"
	ai "bloomify/services/intelligence"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DefaultAIHandler handles AI requests.
type DefaultAIHandler struct {
	svc *ai.DefaultAIService
}

// NewDefaultAIHandler initializes the handler.
func NewDefaultAIHandler(svc *ai.DefaultAIService) *DefaultAIHandler {
	return &DefaultAIHandler{svc: svc}
}

// HandleAIRequest processes a POST request for AI interaction.
func (h *DefaultAIHandler) HandleAIRequest(c *gin.Context) {
	var req models.AIRequest

	// 1. Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format", "details": err.Error()})
		return
	}

	// 2. Process input via AI service
	resp, err := h.svc.ProcessUserInput(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process input", "details": err.Error()})
		return
	}

	// 3. Respond with AIResponse
	c.JSON(http.StatusOK, resp)
}
