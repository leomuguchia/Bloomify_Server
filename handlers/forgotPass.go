package handlers

import (
	"bloomify/services/provider"
	"bloomify/services/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"newPassword"`
	SessionID   string `json:"sessionID"`
}

func ResetUserPasswordHandler(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := userService.ResetPassword(req.Email, req.OTP, req.NewPassword, req.SessionID)
	if err != nil {
		if otpErr, ok := err.(user.OTPPendingError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verification required",
				"code":      100,
				"sessionID": otpErr.SessionID,
			})
			return
		}
		if npErr, ok := err.(user.NewPasswordRequiredError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verified. New password required.",
				"code":      101,
				"sessionID": npErr.SessionID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been successfully reset. Please sign in with your new password."})
}

func ResetProviderPasswordHandler(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := providerService.ResetPassword(req.Email, req.OTP, req.NewPassword, req.SessionID)
	if err != nil {
		if otpErr, ok := err.(provider.OTPPendingError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verification required",
				"code":      100,
				"sessionID": otpErr.SessionID,
			})
			return
		}
		if npErr, ok := err.(provider.NewPasswordRequiredError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "OTP verified. New password required.",
				"code":      101,
				"sessionID": npErr.SessionID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been successfully reset. Please sign in with your new password."})
}
