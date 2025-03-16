package handlers

import (
	"bloomify/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// VerifyOTPHandler verifies the OTP and updates the auth session.
func VerifyOTPHandler(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId" binding:"required"`
		OTP       string `json:"otp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Verify the OTP using the utility function.
	parts := strings.Split(req.SessionID, ":")
	if len(parts) != 2 {
		c.JSON(400, gin.H{"error": "Invalid session ID format"})
		return
	}
	userID := parts[0]
	deviceID := parts[1]
	if err := utils.VerifyDeviceOTPRecord(userID, deviceID, req.OTP); err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	// Retrieve the existing auth session.
	sessionClient := utils.GetAuthCacheClient()
	authSession, err := utils.GetAuthSession(sessionClient, req.SessionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve auth session"})
		return
	}

	// Mark the session as verified.
	authSession.Status = "otp_verified"
	if err := utils.SaveAuthSession(sessionClient, req.SessionID, *authSession); err != nil {
		c.JSON(500, gin.H{"error": "Failed to update auth session"})
		return
	}

	c.JSON(200, gin.H{"message": "OTP verified successfully", "sessionId": req.SessionID})
}
