package provider

import (
	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"

	"github.com/gin-gonic/gin"
)

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo providerRepo.ProviderRepository
}
type ProviderService interface {
	// Core provider methods.
	RegisterBasic(basicReq models.ProviderBasicRegistrationData, device models.Device) (sessionID string, status int, err error)
	VerifyOTP(sessionID string, deviceID string, providedOTP string) (status int, err error)
	VerifyKYP(sessionID string, kypData models.KYPVerificationData) (status int, err error)
	FinalizeRegistration(sessionID string, catalogueData models.ServiceCatalogue) (*models.ProviderAuthResponse, error)

	AuthenticateProvider(email, password string, currentDevice models.Device, providedSessionID string) (*models.ProviderAuthResponse, error)
	RevokeProviderAuthToken(providerID, deviceID string) error
	GetProviderByID(c *gin.Context, id string) (*models.Provider, error)
	GetProviderByEmail(c *gin.Context, email string) (*models.Provider, error)
	UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error)
	UpdateProviderPassword(providerID, currentPassword, newPassword, currentDeviceID string) (*models.Provider, error)
	DeleteProvider(id string) error
	AdvanceVerifyProvider(c *gin.Context, id string, advReq AdvanceVerifyRequest) (*models.Provider, error)
	SetupTimeslots(c *gin.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error)
	GetTimeslots(c *gin.Context, providerID string) ([]models.TimeSlot, error)
	DeleteTimeslot(c *gin.Context, providerID string, timeslotID string) (*models.ProviderTimeslotDTO, error)
	GetAllProviders() ([]models.Provider, error)
	GetProviderDevices(providerID string) ([]models.Device, error)
	SignOutOtherDevices(providerID, currentDeviceID string) error
	ResetPassword(email, providedOTP, newPassword, providedSessionID string) error

	EnableSubscription(providerID string) error
	UpdateSubscriptionSettings(providerID string, settings models.SubscriptionModel) error
	GetSubscriptionHistory(providerID string) ([]models.SubscriptionBooking, error)
}
