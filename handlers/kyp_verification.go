package handlers

import (
	"bloomify/services/provider"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// KYPHTTPReq represents the HTTP request payload for KYP verification.
type KYPHTTPReq struct {
	GovID     string `json:"gov_id" binding:"required"`     // Government ID document reference or scan URL
	Selfie    string `json:"selfie" binding:"required"`     // Selfie image reference or URL
	LegalName string `json:"legal_name" binding:"required"` // Legal name as provided by the user
}

// KYPHTTPResp represents the HTTP response payload for KYP verification.
type KYPHTTPResp struct {
	Verified         bool   `json:"verified"`          // Verification status
	VerificationCode string `json:"verification_code"` // Cryptographic code returned by the service
	Message          string `json:"message"`           // Message from the verification process
	Timestamp        int64  `json:"timestamp"`         // Unix timestamp when verified
}

// KYPVerificationHandler handles HTTP POST requests for KYP verification.
func KYPVerificationHandler(c *gin.Context) {
	logger := zap.L()

	// Bind the JSON payload to our request struct.
	var reqPayload KYPHTTPReq
	if err := c.ShouldBindJSON(&reqPayload); err != nil {
		logger.Error("Invalid JSON payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload: " + err.Error()})
		return
	}

	// Instantiate the KYP verification service.
	verificationService := provider.NewKYPVerificationService()

	// Build the service request.
	kypReq := provider.KYPRequest{
		GovID:     reqPayload.GovID,
		Selfie:    reqPayload.Selfie,
		LegalName: reqPayload.LegalName,
	}

	// Call the verification service.
	kypResp, err := verificationService.VerifyKYP(kypReq)
	if err != nil {
		logger.Error("KYP verification failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build and send the HTTP response.
	respPayload := KYPHTTPResp{
		Verified:         kypResp.Verified,
		VerificationCode: kypResp.VerificationCode,
		Message:          kypResp.Message,
		Timestamp:        kypResp.Timestamp,
	}

	// Optionally set a custom header with the current time.
	c.Header("X-Response-Time", time.Now().Format(time.RFC3339))
	c.JSON(http.StatusOK, respPayload)
}
