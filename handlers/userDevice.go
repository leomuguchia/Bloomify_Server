package handlers

import (
	"bloomify/models"
	"bloomify/services/admin"
	"bloomify/services/provider"
	"bloomify/services/user"
	"bloomify/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type UserHandler struct {
	UserService     user.UserService
	ProviderService provider.ProviderService
	AdminService    admin.AdminService
}

type TrustedProvidersUpdateRequest struct {
	Action           string                   `json:"action"`           // "add" or "remove"
	TrustedProviders []models.TrustedProvider `json:"trustedProviders"` // must match your model
}

func NewUserHandler(userService user.UserService, providerService provider.ProviderService, adminService admin.AdminService) *UserHandler {
	return &UserHandler{
		UserService:     userService,
		ProviderService: providerService,
		AdminService:    adminService,
	}
}
func (h *UserHandler) GetUserDevicesHandler(c *gin.Context) {
	// Retrieve userID from context (set by JWTAuthUserMiddleware)
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return
	}

	devices, err := h.UserService.GetUserDevices(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *UserHandler) SignOutOtherUserDevicesHandler(c *gin.Context) {
	// Retrieve userID from context (set by JWTAuthUserMiddleware)
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return
	}

	currentDeviceID, exists := c.Get("deviceID")
	if !exists || currentDeviceID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Device ID not found in context"})
		return
	}

	err := h.UserService.SignOutOtherDevices(userID, currentDeviceID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signed out of other devices successfully"})
}

func (h *UserHandler) UpdateFCMTokenHandler(c *gin.Context) {
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		FCMToken string `json:"fcmToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Log the update attempt
	utils.Logger.Info("Updating FCM token", zap.String("userID", userID), zap.String("newFCMToken", req.FCMToken))

	updatedUser, err := h.UserService.UpdateUser(models.User{
		ID:       userID,
		FCMToken: req.FCMToken,
	})
	if err != nil {
		utils.Logger.Error("Failed to update FCM token", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	utils.Logger.Info("Successfully updated FCM token", zap.String("userID", userID))
	c.JSON(http.StatusOK, updatedUser)
}

func (h *UserHandler) UpdateTrustedProviders(c *gin.Context) {
	rawUserID, exists := c.Get("userID")
	if !exists || rawUserID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := rawUserID.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	var req TrustedProvidersUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if userID == "" || (req.Action != "add" && req.Action != "remove") || len(req.TrustedProviders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid parameters"})
		return
	}

	switch req.Action {
	case "add":
		user := models.User{
			ID:               userID,
			TrustedProviders: req.TrustedProviders,
		}
		updated, err := h.UserService.UpdateUser(user)
		if err != nil {
			utils.GetLogger().Error("Failed to add trusted providers", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add trusted providers"})
			return
		}
		c.JSON(http.StatusOK, updated)

	case "remove":
		var providerMatches []interface{}
		for _, tp := range req.TrustedProviders {
			providerMatches = append(providerMatches, bson.M{"providerId": tp.ProviderID})
		}

		updated, err := h.UserService.RemoveFromUser(userID, "trustedProviders", providerMatches)
		if err != nil {
			utils.GetLogger().Error("Failed to remove trusted providers", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove trusted providers"})
			return
		}
		c.JSON(http.StatusOK, updated)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported action"})
	}
}
