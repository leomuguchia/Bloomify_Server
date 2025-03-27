package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AIRecommendRequest is the expected input structure for AI recommendations.
type AIRecommendRequest struct {
	Input  string `json:"input"`
	UserID uint   `json:"user_id"`
}

// AIRecommendResponse represents the response from the AI microservice for recommendations.
type AIRecommendResponse struct {
	Service        string `json:"service"`
	Recommendation string `json:"recommendation"`
}

// AISuggestRequest represents the request for time-slot suggestions.
type AISuggestRequest struct {
	Input    string `json:"input"`
	Duration int    `json:"duration"`
}

// AISuggestResponse represents the suggested time slot.
type AISuggestResponse struct {
	SuggestedSlot string `json:"suggested_slot"`
}

// AutoBookRequest represents a request to auto-book a service.
type AutoBookRequest struct {
	Input string `json:"input"`
}

// getAIServiceURL retrieves the AI service URL from configuration.
// If not defined, it falls back to a default URL based on the provided endpoint.
func getAIServiceURL(defaultEndpoint string) string {
	aiURL := viper.GetString("AI_SERVICE_URL")
	if aiURL == "" {
		return fmt.Sprintf("http://ai-service:8000/%s", defaultEndpoint)
	}
	return replaceEndpoint(aiURL, defaultEndpoint)
}

// replaceEndpoint adjusts the given AI service URL to use a different endpoint.
func replaceEndpoint(baseURL, newEndpoint string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Sprintf("http://ai-service:8000/%s", newEndpoint)
	}
	u.Path = path.Join("/", newEndpoint)
	return u.String()
}

// Define a package-level HTTP client for AI service calls.
var aiHTTPClient = &http.Client{Timeout: 5 * time.Second}

// AIRecommendHandler handles recommendation requests by forwarding them to the AI microservice.
func AIRecommendHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var req AIRecommendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid AI recommend request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// If userID is missing in JSON, attempt to extract it from context.
	if req.UserID == 0 {
		if uid, exists := c.Get("userID"); exists {
			if uidStr, ok := uid.(string); ok {
				if parsedID, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
					req.UserID = uint(parsedID)
				} else {
					logger.Warn("Failed to convert userID", zap.Error(err))
				}
			}
		}
	}

	payload, err := json.Marshal(req)
	if err != nil {
		logger.Error("Failed to marshal AI recommend request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	aiURL := getAIServiceURL("recommend")
	resp, err := aiHTTPClient.Post(aiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Failed to call AI service", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reach AI service: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("AI service returned non-OK status", zap.Int("status", resp.StatusCode))
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			logger.Error("Failed to decode error response from AI service", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "AI service error",
			"message": errResp,
		})
		return
	}

	var aiResp AIRecommendResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		logger.Error("Failed to decode AI service response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode AI response"})
		return
	}
	c.JSON(http.StatusOK, aiResp)
}

// AISuggestHandler handles requests for time-slot suggestions.
func AISuggestHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var req AISuggestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid AI suggest request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		logger.Error("Failed to marshal AI suggest request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	aiURL := getAIServiceURL("suggest")
	resp, err := aiHTTPClient.Post(aiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Failed to call AI suggest service", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reach AI service: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("AI suggest service returned non-OK status", zap.Int("status", resp.StatusCode))
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			logger.Error("Failed to decode error response from AI suggest service", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "AI service error",
			"message": errResp,
		})
		return
	}

	var aiResp AISuggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		logger.Error("Failed to decode AI suggest response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode AI response"})
		return
	}
	c.JSON(http.StatusOK, aiResp)
}

// AutoBookHandler processes auto-book requests by forwarding them to the AI microservice.
func AutoBookHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var req AutoBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid auto-book request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		logger.Error("Failed to marshal auto-book request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	aiURL := getAIServiceURL("auto-book")
	resp, err := aiHTTPClient.Post(aiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Failed to call AI auto-book service", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reach AI service: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("AI auto-book service returned non-OK status", zap.Int("status", resp.StatusCode))
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			logger.Error("Failed to decode error response from AI auto-book service", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "AI service error",
			"message": errResp,
		})
		return
	}

	var bookingResp models.Booking
	if err := json.NewDecoder(resp.Body).Decode(&bookingResp); err != nil {
		logger.Error("Failed to decode AI auto-book response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode AI response"})
		return
	}
	c.JSON(http.StatusOK, bookingResp)
}
