package provider

import (
	providerRepo "bloomify/database/repository/provider"
	recordsRepo "bloomify/database/repository/records"
	timeslotRepo "bloomify/database/repository/timeslot"
	"bloomify/models"
	"context"
)

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo        providerRepo.ProviderRepository
	Timeslot    timeslotRepo.TimeSlotRepository
	RecordsRepo recordsRepo.HistoricalRecordRepository
}

type ProviderService interface {
	RegisterBasic(basicReq models.ProviderBasicRegistrationData, device models.Device) (sessionID string, status int, err error)
	VerifyOTP(sessionID string, deviceID string, providedOTP string) (status int, err error)
	VerifyKYP(sessionID string, kypData models.KYPVerificationData) (status int, err error)
	FinalizeRegistration(sessionID string, catalogueData models.ServiceCatalogue) (*models.ProviderAuthResponse, error)

	AuthenticateProvider(email, password string, currentDevice models.Device, providedSessionID string) (*models.ProviderAuthResponse, error)
	RevokeProviderAuthToken(providerID, deviceID string) error
	GetProviderByID(c context.Context, id string, fullAccess bool) (*models.Provider, error)
	GetProviderByEmail(c context.Context, email string, fullAcess bool) (*models.Provider, error)
	UpdateProvider(c context.Context, id string, updates map[string]interface{}) (*models.Provider, error)
	UpdateProviderPassword(providerID, currentPassword, newPassword, currentDeviceID string) (*models.Provider, error)
	DeleteProvider(id string) error
	AdvanceVerifyProvider(c context.Context, id string, advReq AdvanceVerifyRequest, fullAccess bool) (*models.Provider, error)

	// Timeslot management (now all require date)
	SetupTimeslots(c context.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error)
	GetTimeslots(c context.Context, providerID, date string) ([]models.TimeSlot, error)
	GetTimeslot(c context.Context, providerID, timeslotID, date string) (*models.TimeSlot, error)
	DeleteTimeslot(c context.Context, providerID, timeslotID, date string) (*models.ProviderTimeslotDTO, error)

	// Other methods...
	GetAllProviders() ([]models.Provider, error)
	GetProviderDevices(providerID string) ([]models.Device, error)
	SignOutOtherDevices(providerID, currentDeviceID string) error
	ResetPassword(email, providedOTP, newPassword, providedSessionID string) error

	// Subscription management...
	EnableSubscription(providerID string) error
	UpdateSubscriptionSettings(providerID string, settings models.SubscriptionModel) error
	GetSubscriptionHistory(providerID string) ([]models.SubscriptionBooking, error)

	// Historical records...
	GetHistoricalRecords(c context.Context, providerID string) ([]models.HistoricalRecord, error)
	AddHistoricalRecord(c context.Context, record models.HistoricalRecord) (string, error)
	DeleteHistoricalRecord(c context.Context, recordID string) error
}
