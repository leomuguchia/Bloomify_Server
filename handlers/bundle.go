package handlers

import "github.com/gin-gonic/gin"

// HandlerBundle groups all your endpoint handlers into one struct.
type HandlerBundle struct {

	// Provider endpoints
	GetProviderByIDHandler      gin.HandlerFunc
	GetProviderByEmailHandler   gin.HandlerFunc
	RegisterProviderHandler     gin.HandlerFunc
	UpdateProviderHandler       gin.HandlerFunc
	DeleteProviderHandler       gin.HandlerFunc
	AuthenticateProviderHandler gin.HandlerFunc
	KYPVerificationHandler      gin.HandlerFunc

	// Booking endpoints
	InitiateSession gin.HandlerFunc
	UpdateSession   gin.HandlerFunc
	ConfirmBooking  gin.HandlerFunc

	// AI endpoints (individual functions, as you don't have a dedicated AI handler)
	AIRecommendHandler gin.HandlerFunc
	AISuggestHandler   gin.HandlerFunc
	AutoBookHandler    gin.HandlerFunc

	// User endpoints
	RegisterUserHandler     gin.HandlerFunc
	AuthenticateUserHandler gin.HandlerFunc
	GetUserByIDHandler      gin.HandlerFunc
	GetUserByEmailHandler   gin.HandlerFunc
	UpdateUserHandler       gin.HandlerFunc
	DeleteUserHandler       gin.HandlerFunc
}
