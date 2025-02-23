package handlers

import (
	"net/http"

	"bloomify/database/repository"
	"bloomify/models"
	"bloomify/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Global instance of AuthService using our DefaultAuthService.
var authService services.AuthService = &services.DefaultAuthService{
	UserRepo: &repository.GormUserRepo{},
}

// RegisterHandler handles user registration.
func RegisterHandler(c *gin.Context) {
	logger := getLogger(c)

	var req models.User
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	registeredUser, err := authService.RegisterUser(req)
	if err != nil {
		logger.Error("User registration failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, registeredUser)
}

// LoginHandler handles user login and returns access and refresh tokens.
func LoginHandler(c *gin.Context) {
	logger := getLogger(c)

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// im hoping everything works, but cant test until i finish the remainign code. we have done authentication, database abstraction, unified booking engine...now lets go to

	accessToken, refreshToken, err := authService.LoginUser(req.Email, req.Password)
	if err != nil {
		logger.Error("Login failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Login failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

// RefreshTokenHandler issues new tokens based on a valid refresh token.
func RefreshTokenHandler(c *gin.Context) {
	logger := getLogger(c)

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid refresh token request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	accessToken, newRefreshToken, err := authService.RefreshToken(req.RefreshToken)
	if err != nil {
		logger.Error("Token refresh failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token refresh failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": newRefreshToken})
}
