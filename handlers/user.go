package handlers

import (
	"net/http"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterUserHandler orchestrates the three-step registration process.
// "basic": Initiates registration (returns code 100 on success).
// "otp": Verifies the OTP (returns code 101 on success).
// "preferences": Finalizes registration (returns AuthResponse, code 102).
func (h *UserHandler) RegisterUserHandler(c *gin.Context) {
	logger := utils.GetLogger()

	// Extract device details from context.
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

	// Bind request body.
	var req models.UserRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Step {
	case "basic":
		if req.BasicData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing basic registration data"})
			return
		}
		sessionID, code, err := h.UserService.InitiateRegistration(*req.BasicData, device)
		if err != nil {
			logger.Error("Basic registration failed", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Success: code 100 indicates OTP pending.
		c.JSON(http.StatusAccepted, gin.H{"sessionID": sessionID, "status": code})

	case "otp":
		if req.SessionID == "" || req.OTP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID or OTP for verification"})
			return
		}
		code, err := h.UserService.VerifyRegistrationOTP(req.SessionID, device.DeviceID, req.OTP)
		if err != nil {
			logger.Error("OTP verification failed", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Success: code 101 indicates OTP verified.
		c.JSON(http.StatusOK, gin.H{"sessionID": req.SessionID, "status": code})

	case "preferences":
		if req.SessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID for finalizing registration"})
			return
		}
		if len(req.Preferences) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Preferences are required to finalize registration"})
			return
		}
		if !req.EmailUpdates {
			req.EmailUpdates = false // Default to false if not provided
		}
		authResp, err := h.UserService.FinalizeRegistration(req.SessionID, req.Preferences, req.EmailUpdates)
		if err != nil {
			logger.Error("Finalizing registration failed", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Success: code 102 indicates registration complete.
		c.JSON(http.StatusCreated, gin.H{"auth": authResp, "code": 102})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration step"})
	}
}

// AuthenticateUserHandler handles user sign-in with a step-based approach.
// First request (without sessionID): Initiates authentication (returns code 100 if OTP required)
// Subsequent request (with sessionID): Continues authentication (OTP verification or completion)
func (h *UserHandler) AuthenticateUserHandler(c *gin.Context) {
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

	// Handle different cases based on presence of sessionID and OTP
	switch {
	case req.SessionID == "":
		// Initial authentication request
		if req.Method == "password" && req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password is required for password authentication"})
			return
		}

		authResp, sessionID, code, err := h.UserService.InitiateAuthentication(req.Email, req.Method, req.Password, currentDevice)
		if err != nil {
			if code == 100 { // OTP required
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":     "OTP verification required",
					"code":      code,
					"sessionID": sessionID,
				})
				return
			}
			logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if authResp != nil {
			// Successful authentication without OTP
			c.JSON(http.StatusOK, authResp)
			return
		}

		// Shouldn't reach here if logic is correct
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected authentication state"})

	case req.SessionID != "" && req.OTP == "":
		// Check authentication status
		status, err := h.UserService.CheckAuthenticationStatus(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status})

	case req.SessionID != "" && req.OTP != "":
		// OTP verification
		authResp, err := h.UserService.VerifyAuthenticationOTP(req.SessionID, req.OTP, currentDevice)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, authResp)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters"})
	}
}

// RevokeUserAuthTokenHandler handles token revocation for a user.
// It requires the user ID in the URL parameter and uses the device details from context.
func (h *UserHandler) RevokeUserAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	userID := c.Param("id")

	// Extract device ID from context.
	deviceID, ok := c.Get("deviceID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing device details: X-Device-ID"})
		return
	}

	// Call the service to revoke the token for this specific device.
	if err := h.UserService.RevokeUserAuthToken(userID, deviceID.(string)); err != nil {
		logger.Error("Revoke token error", zap.String("userID", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}
