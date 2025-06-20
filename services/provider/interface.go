package provider

import (
	providerRepo "bloomify/database/repository/provider"
	recordsRepo "bloomify/database/repository/records"
	schedulerRepo "bloomify/database/repository/scheduler"
	timeslotRepo "bloomify/database/repository/timeslot"
	"bloomify/models"
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

// DefaultProviderService is the production implementation.
type DefaultProviderService struct {
	Repo          providerRepo.ProviderRepository
	Timeslot      timeslotRepo.TimeSlotRepository
	RecordsRepo   recordsRepo.HistoricalRecordRepository
	AsynqClient   *asynq.Client
	SchedulerRepo schedulerRepo.SchedulerRepository
}

func NewDefaultProviderService(
	repo providerRepo.ProviderRepository,
	timeslot timeslotRepo.TimeSlotRepository,
	recordsRepo recordsRepo.HistoricalRecordRepository,
	asynqClient *asynq.Client,
	schedulerRepo schedulerRepo.SchedulerRepository,
) (*DefaultProviderService, error) {
	if repo == nil || timeslot == nil || recordsRepo == nil || asynqClient == nil {
		return nil, fmt.Errorf("provider service initialization error: one or more dependencies are nil")
	}

	return &DefaultProviderService{
		Repo:          repo,
		Timeslot:      timeslot,
		RecordsRepo:   recordsRepo,
		AsynqClient:   asynqClient,
		SchedulerRepo: schedulerRepo,
	}, nil
}

type ProviderService interface {
	// Registration
	RegisterBasic(basicReq models.ProviderBasicRegistrationData, device models.Device) (sessionID string, status int, err error)
	VerifyOTP(sessionID string, deviceID string, providedOTP string) (status int, err error)
	VerifyKYP(sessionID string, kypData models.KYPVerificationData) (status int, err error)
	FinalizeRegistration(sessionID string, catalogueData models.ServiceCatalogue) (*models.ProviderAuthResponse, error)

	// Authentication
	InitiateProviderAuthentication(email, method, password string, currentDevice models.Device) (*models.ProviderAuthResponse, string, int, error)
	CheckProviderAuthenticationStatus(sessionID string) (string, error)
	VerifyProviderAuthenticationOTP(sessionID, otp string, currentDevice models.Device) (*models.ProviderAuthResponse, error)
	RevokeProviderAuthToken(providerID, deviceID string) error

	// Account Management
	GetProviderByID(c context.Context, id string, fullAccess bool) (*models.Provider, error)
	GetProviderByEmail(c context.Context, email string, fullAcess bool) (*models.Provider, error)
	UpdateProvider(c context.Context, id string, updates map[string]interface{}) (*models.Provider, error)
	UpdateProviderPassword(providerID, currentPassword, newPassword, currentDeviceID string) (*models.Provider, error)
	DeleteProvider(id string) error
	AdvanceVerifyProvider(c context.Context, id string, advReq AdvanceVerifyRequest, fullAccess bool) (*models.Provider, error)

	// Timeslot Management
	SetupTimeslots(c context.Context, providerID string, req models.SetupTimeslotsRequest) (*models.ProviderTimeslotDTO, error)
	GetTimeslots(c context.Context, providerID, date string) ([]models.TimeSlot, error)
	GetTimeslot(c context.Context, providerID, timeslotID, date string) (*models.TimeSlot, error)
	DeleteTimeslot(c context.Context, providerID, timeslotID, date string) (*models.ProviderTimeslotDTO, error)
	VerifyBooking(ctx context.Context, providerID string, date string, bookingID string) (*models.Booking, error)

	// Other methods
	GetAllProviders() ([]models.Provider, error)
	GetProviderDevices(providerID string) ([]models.Device, error)
	SignOutOtherDevices(providerID, currentDeviceID string) error
	ResetPassword(email, providedOTP, newPassword, providedSessionID string) error

	// Subscription Management
	EnableSubscription(providerID string) error
	UpdateSubscriptionSettings(providerID string, settings models.SubscriptionModel) error
	GetSubscriptionHistory(providerID string) ([]models.SubscriptionBooking, error)

	// Historical Records
	GetHistoricalRecords(c context.Context, providerID string) ([]models.HistoricalRecord, error)
	AddHistoricalRecord(c context.Context, record models.HistoricalRecord) (string, error)
	DeleteHistoricalRecord(c context.Context, recordID string) error
}
