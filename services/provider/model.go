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

// ProviderService defines the interface for provider-related operations.
type ProviderService interface {
	RegisterProvider(provider models.Provider) (*ProviderAuthResponse, error)
	AuthenticateProvider(email, password string, currentDevice models.Device, providedSessionID string) (*ProviderAuthResponse, error)
	RevokeProviderAuthToken(providerID string) error
	GetProviderByID(c *gin.Context, id string) (*models.Provider, error)
	GetProviderByEmail(c *gin.Context, email string) (*models.Provider, error)
	UpdateProvider(c *gin.Context, id string, updates map[string]interface{}) (*models.Provider, error)
	DeleteProvider(id string) error
	AdvanceVerifyProvider(c *gin.Context, id string, advReq AdvanceVerifyRequest) (*models.Provider, error)
	SetupTimeslots(c *gin.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error)
	GetTimeslots(c *gin.Context, providerID string) ([]models.TimeSlot, error)
	DeleteTimeslot(c *gin.Context, providerID string, timeslotID string) (*models.ProviderTimeslotDTO, error)
	GetAllProviders() ([]models.Provider, error)
}
