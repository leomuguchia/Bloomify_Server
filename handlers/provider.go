package handlers

import (
	"net/http"
	"time"

	"bloomify/models"
	"bloomify/services/admin"
	"bloomify/services/provider"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProviderHandler struct {
	Service      provider.ProviderService
	AdminService admin.AdminService
}

func NewProviderHandler(ps provider.ProviderService, as admin.AdminService) *ProviderHandler {
	return &ProviderHandler{
		Service:      ps,
		AdminService: as,
	}
}

// RegisterProviderHandler orchestrates the multi‑step registration process.
func (h *ProviderHandler) RegisterProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()

	deviceID, exists := c.Get("deviceID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}
	deviceName, exists := c.Get("deviceName")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-Name"})
		return
	}
	deviceIP, _ := c.Get("deviceIP")
	deviceLocation, _ := c.Get("deviceLocation")
	device := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
		Creator:    true,
	}

	var req models.ProviderRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	switch req.Step {
	case "basic":
		// Step 1: Basic Registration + OTP Initiation.
		if req.BasicData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing basic registration data"})
			return
		}
		sessionID, status, err := h.Service.RegisterBasic(*req.BasicData, device)
		if err != nil {
			logger.Error("Failed in basic registration", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Basic registration failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessionID": sessionID, "status": status})
	case "otp":
		// Step 1.5: OTP Verification.
		if req.SessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID for OTP verification"})
			return
		}
		if req.OTP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing OTP"})
			return
		}
		status, err := h.Service.VerifyOTP(req.SessionID, device.DeviceID, req.OTP)
		if err != nil {
			logger.Error("Failed in OTP verification", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessionID": req.SessionID, "status": status})
	case "kyp":
		// Step 2: KYP Verification.
		if req.SessionID == "" || req.KYPData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID or KYP data"})
			return
		}
		status, err := h.Service.VerifyKYP(req.SessionID, *req.KYPData)
		if err != nil {
			logger.Error("Failed in KYP verification", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "KYP verification failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessionID": req.SessionID, "status": status})
	case "catalogue":
		// Step 3: Service Catalogue & Finalization.
		if req.SessionID == "" || req.ServiceCatalogue == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID or service catalogue data"})
			return
		}
		providerAuthResp, err := h.Service.FinalizeRegistration(req.SessionID, *req.ServiceCatalogue)
		if err != nil {
			logger.Error("Failed to finalize registration", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Registration finalization failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, providerAuthResp)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration step"})
	}
}

// AuthenticateProviderHandler handles provider sign-in with a step-based approach.
// First request (without sessionID): Initiates authentication (returns code 100 if OTP required)
// Subsequent request (with sessionID): Continues authentication (OTP verification or completion)
func (h *ProviderHandler) AuthenticateProviderHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Extract device details from context
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device ID"})
		return
	}
	deviceName, _ := c.Get("deviceName")
	deviceIP, _ := c.Get("deviceIP")
	deviceLocation, _ := c.Get("deviceLocation")

	currentDevice := models.Device{
		DeviceID:   deviceID.(string),
		DeviceName: deviceName.(string),
		IP:         deviceIP.(string),
		Location:   deviceLocation.(string),
		LastLogin:  time.Now(),
	}

	// Parse request body
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password"`
		Method    string `json:"method" binding:"required"`
		SessionID string `json:"sessionID"`
		OTP       string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 1: Initial login attempt (no sessionID)
	if req.SessionID == "" {
		if req.Method == "password" && req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required for password authentication"})
			return
		}

		authResp, sessionID, code, err := h.Service.InitiateProviderAuthentication(req.Email, req.Method, req.Password, currentDevice)

		// OTP required
		if code == 100 && sessionID != "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verification required",
				"code":      code,
				"sessionID": sessionID,
			})
			return
		}

		// Actual error
		if err != nil {
			logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Success without OTP
		if authResp != nil {
			c.JSON(http.StatusOK, authResp)
			return
		}

		// Unexpected state
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected authentication state"})
		return
	}

	// Step 2a: SessionID present, but no OTP — check status
	if req.SessionID != "" && req.OTP == "" {
		status, err := h.Service.CheckProviderAuthenticationStatus(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status})
		return
	}

	// Step 2b: SessionID and OTP present — verify
	if req.SessionID != "" && req.OTP != "" {
		authResp, err := h.Service.VerifyProviderAuthenticationOTP(req.SessionID, req.OTP, currentDevice)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, authResp)
		return
	}

	// Final fallback
	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
}

func (h *ProviderHandler) RevokeProviderAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	providerID := c.Param("id")

	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}

	if err := h.Service.RevokeProviderAuthToken(providerID, deviceID.(string)); err != nil {
		logger.Error("Failed to revoke provider auth token", zap.String("id", providerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke auth token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}
