// File: bloomify/handlers/handlerBundle.go
package handlers

import (
	providerRepoPkg "bloomify/database/repository/provider"
	userRepoPkg "bloomify/database/repository/user"

	"github.com/gin-gonic/gin"
)

// HandlerBundle groups all your endpoint handlers into one struct.
type HandlerBundle struct {
	ProviderRepo providerRepoPkg.ProviderRepository
	UserRepo     userRepoPkg.UserRepository

	// Provider endpoints
	GetProviderByIDHandler         gin.HandlerFunc
	GetProviderByEmailHandler      gin.HandlerFunc
	RegisterProviderHandler        gin.HandlerFunc
	UpdateProviderHandler          gin.HandlerFunc
	DeleteProviderHandler          gin.HandlerFunc
	AuthenticateProviderHandler    gin.HandlerFunc
	KYPVerificationHandler         gin.HandlerFunc
	AdvanceVerifyProviderHandler   gin.HandlerFunc
	RevokeProviderAuthTokenHandler gin.HandlerFunc

	// Booking endpoints
	InitiateSession gin.HandlerFunc
	UpdateSession   gin.HandlerFunc
	ConfirmBooking  gin.HandlerFunc

	// AI endpoints
	AIRecommendHandler gin.HandlerFunc
	AISuggestHandler   gin.HandlerFunc
	AutoBookHandler    gin.HandlerFunc

	// User endpoints
	RegisterUserHandler        gin.HandlerFunc
	AuthenticateUserHandler    gin.HandlerFunc
	GetUserByIDHandler         gin.HandlerFunc
	GetUserByEmailHandler      gin.HandlerFunc
	UpdateUserHandler          gin.HandlerFunc
	DeleteUserHandler          gin.HandlerFunc
	RevokeUserAuthTokenHandler gin.HandlerFunc
}
