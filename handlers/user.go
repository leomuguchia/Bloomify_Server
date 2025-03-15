package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var userService user.UserService

func SetUserService(us user.UserService) {
	userService = us
}

func RegisterUserHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResp, err := userService.RegisterUser(reqUser)
	if err != nil {
		logger.Error("Registration error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, authResp)
}

func AuthenticateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResp, err := userService.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		logger.Error("Authentication error", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, authResp)
}

func GetUserByIDHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	usr, err := userService.GetUserByID(id)
	if err != nil {
		logger.Error("User not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// GetUserByEmailHandler handles GET /users/email/:email.
func GetUserByEmailHandler(c *gin.Context) {
	logger := utils.GetLogger()
	email := c.Param("email")
	usr, err := userService.GetUserByEmail(email)
	if err != nil {
		logger.Error("User not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usr)
}

func UpdateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")

	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reqUser.ID = id

	updatedUser, err := userService.UpdateUser(reqUser)
	if err != nil {
		logger.Error("Update error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUserHandler handles DELETE /users/delete/:id.
func DeleteUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.DeleteUser(id); err != nil {
		logger.Error("Delete error", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// RevokeUserAuthTokenHandler handles DELETE /users/revoke/:id.
func RevokeUserAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.RevokeUserAuthToken(id); err != nil {
		logger.Error("Revoke token error", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}

// UpdateUserPreferencesHandler handles PUT /users/preferences/:id.
func UpdateUserPreferencesHandler(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Preferences []string `json:"preferences" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := userService.UpdateUserPreferences(userID, req.Preferences); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// UpdateUserPasswordHandler handles PUT /users/password/:id.
// It expects a JSON payload with "currentPassword" and "newPassword".
func UpdateUserPasswordHandler(c *gin.Context) {
	logger := utils.GetLogger()
	userID := c.Param("id")

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid update password request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser, err := userService.UpdateUserPassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		logger.Error("Failed to update user password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
