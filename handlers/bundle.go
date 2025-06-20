package handlers

import (
	providerRepoPkg "bloomify/database/repository/provider"
	userRepoPkg "bloomify/database/repository/user"

	"github.com/gin-gonic/gin"
)

type HandlerBundle struct {
	ProviderRepo providerRepoPkg.ProviderRepository
	UserRepo     userRepoPkg.UserRepository

	// Provider endpoints (unified registration endpoint now)
	GetProviderByIDHandler         gin.HandlerFunc
	GetProviderByEmailHandler      gin.HandlerFunc
	RegisterProviderHandler        gin.HandlerFunc // Unified registration (basic/OTP/KYP/catalogue)
	UpdateProviderHandler          gin.HandlerFunc
	DeleteProviderHandler          gin.HandlerFunc
	AuthenticateProviderHandler    gin.HandlerFunc
	AdvanceVerifyProviderHandler   gin.HandlerFunc
	UpdateProviderPasswordHandler  gin.HandlerFunc
	RevokeProviderAuthTokenHandler gin.HandlerFunc
	SetupTimeslotsHandler          gin.HandlerFunc
	GetTimeslotsHandler            gin.HandlerFunc
	DeleteTimeslotHandler          gin.HandlerFunc
	ProviderLegalDocumentation     gin.HandlerFunc
	VerifyBooking                  gin.HandlerFunc

	// Provider device endpoints
	GetProviderDevicesHandler          gin.HandlerFunc
	SignOutOtherProviderDevicesHandler gin.HandlerFunc
	UpdateProviderFCMTokenHandler      gin.HandlerFunc

	// Booking endpoints
	InitiateSession      gin.HandlerFunc
	UpdateSession        gin.HandlerFunc
	ConfirmBooking       gin.HandlerFunc
	CancelSession        gin.HandlerFunc
	GetAvailableServices gin.HandlerFunc
	GetServiceByID       gin.HandlerFunc
	GetDirections        gin.HandlerFunc
	GetPaymentIntent     gin.HandlerFunc
	MatchNearbyProviders gin.HandlerFunc
	GeocodeAddress       gin.HandlerFunc
	ReverseGeocode       gin.HandlerFunc

	// AI endpoints
	AIChatHandler gin.HandlerFunc
	AISTTHandler  gin.HandlerFunc

	// User endpoints
	RegisterUserHandler        gin.HandlerFunc
	AuthenticateUserHandler    gin.HandlerFunc
	GetUserByIDHandler         gin.HandlerFunc
	GetUserByEmailHandler      gin.HandlerFunc
	UpdateUserHandler          gin.HandlerFunc
	DeleteUserHandler          gin.HandlerFunc
	RevokeUserAuthTokenHandler gin.HandlerFunc
	UpdateUserPasswordHandler  gin.HandlerFunc
	UpdateFCMTokenHandler      gin.HandlerFunc
	UserLegalDocumentation     gin.HandlerFunc
	UpdateSafetyPreferences    gin.HandlerFunc
	UpdateTrustedProviders     gin.HandlerFunc

	// User device endpoints
	GetUserDevicesHandler          gin.HandlerFunc
	SignOutOtherUserDevicesHandler gin.HandlerFunc

	// Password reset endpoints for users
	ResetPasswordHandler gin.HandlerFunc

	// New provider password reset endpoint.
	ResetProviderPasswordHandler gin.HandlerFunc

	// Admin endpoints
	AdminHandler            *AdminHandler
	AdminLegalDocumentation gin.HandlerFunc
	GetAllProvidersHandler  gin.HandlerFunc
	GetAllUsersHandler      gin.HandlerFunc

	// Storage endpoints
	StorageHandler           *StorageHandler
	UploadFileHandler        gin.HandlerFunc
	GetDownloadURLHandler    gin.HandlerFunc
	KYPUploadFileHandler     gin.HandlerFunc
	KYPGetDownloadURLHandler gin.HandlerFunc
}
