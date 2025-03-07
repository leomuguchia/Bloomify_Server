// File: bloomify/handlers/user.go
package handlers

import (
	"net/http"

	"bloomify/models"
	"bloomify/services/user"
	"bloomify/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// userService holds the UserService instance and should be set during app initialization.
var userService user.UserService

// SetUserService sets the user service instance.
func SetUserService(us user.UserService) {
	userService = us
}

// RegisterUserHandler handles POST /users.
// It registers a new user.
func RegisterUserHandler(c *gin.Context) {
	logger := utils.GetLogger()

	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	authResp, err := userService.RegisterUser(reqUser)
	if err != nil {
		logger.Error("Failed to register user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, authResp)
}

// AuthenticateUserHandler handles POST /users/login.
// It verifies the user's email and password.
func AuthenticateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid authentication request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	authResp, err := userService.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		logger.Error("Authentication failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	c.JSON(http.StatusOK, authResp)
}

// GetUserByIDHandler handles GET /users/:id.
// It retrieves a user by its ID.
func GetUserByIDHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	usr, err := userService.GetUserByID(id)
	if err != nil {
		logger.Error("User not found", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// GetUserByEmailHandler handles GET /users/email/:email.
// It retrieves a user by its email.
func GetUserByEmailHandler(c *gin.Context) {
	logger := utils.GetLogger()
	email := c.Param("email")
	usr, err := userService.GetUserByEmail(email)
	if err != nil {
		logger.Error("User not found by email", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, usr)
}

// UpdateUserHandler handles PUT /users/:id.
// It updates an existing user's allowed fields.
func UpdateUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")

	var reqUser models.User
	if err := c.ShouldBindJSON(&reqUser); err != nil {
		logger.Error("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	reqUser.ID = id

	updatedUser, err := userService.UpdateUser(reqUser)
	if err != nil {
		logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUserHandler handles DELETE /users/:id.
// It deletes the user with the given ID.
func DeleteUserHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.DeleteUser(id); err != nil {
		logger.Error("Failed to delete user", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// RevokeUserAuthTokenHandler handles DELETE /users/revoke/:id.
// It revokes the user's auth token by clearing the token hash.
func RevokeUserAuthTokenHandler(c *gin.Context) {
	logger := utils.GetLogger()
	id := c.Param("id")
	if err := userService.RevokeUserAuthToken(id); err != nil {
		logger.Error("Failed to revoke user auth token", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke auth token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Auth token revoked"})
}
