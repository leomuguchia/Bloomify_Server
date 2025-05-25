package booking

import (
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	schedulerRepo "bloomify/database/repository/scheduler"
	timeslotRepo "bloomify/database/repository/timeslot"
	"bloomify/models"
	"bloomify/services/notification"
	"bloomify/services/user"
	"bloomify/utils"

	"go.uber.org/zap"
)

// DefaultSchedulingEngine is our production-grade scheduler.
type DefaultSchedulingEngine struct {
	Repo           schedulerRepo.SchedulerRepository
	PaymentHandler PaymentHandler
	ProviderRepo   providerRepo.ProviderRepository
	TimeslotsRepo  timeslotRepo.TimeSlotRepository
	UserService    user.UserService
	Notification   notification.NotificationService
}

type AvailableSlotsResult struct {
	Slots               []models.AvailableSlot
	AvailabilityError   string                   `json:"availabilityError,omitempty"`
	MaxAvailableDate    string                   `json:"maxAvailableDate,omitempty"`
	SubscriptionAllowed bool                     `json:"subscriptionAllowed"`
	SubscriptionModel   models.SubscriptionModel `json:"subscriptionModel,omitzero"`
}

func (se *DefaultSchedulingEngine) GetWeeklyAvailableSlots(
	provider models.Provider,
	weekIndex int,
) (AvailableSlotsResult, error) {
	logger := utils.GetLogger()
	now := time.Now()

	// 1. Fetch maxDate (as before)
	maxDate, err := se.TimeslotsRepo.GetMaxAvailableDate(provider.ID)
	if err != nil {
		logger.Error("GetWeeklyAvailableSlots: error fetching max available date",
			zap.String("providerID", provider.ID), zap.Error(err))
	}

	// 2. Compute week window & dates
	weekZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := weekZero.AddDate(0, 0, weekIndex*7)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// 3. Gather raw slots
	var raw []models.TimeSlot
	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		daySlots, err := se.TimeslotsRepo.GetAvailableTimeSlots(provider.ID, dateStr)
		if err != nil {
			logger.Error("error fetching timeslots", zap.String("date", dateStr), zap.Error(err))
			continue
		}
		raw = append(raw, daySlots...)
	}
	if len(raw) == 0 {
		return AvailableSlotsResult{
			Slots:             nil,
			AvailabilityError: "No schedule available for the selected provider",
			MaxAvailableDate:  maxDate,
			// subscription flags off by default
			SubscriptionAllowed: false,
		}, nil
	}

	// 4. Enrich + build
	enriched := EnrichTimeslots(raw, provider.ServiceCatalogue, logger)
	if len(enriched) == 0 {
		return AvailableSlotsResult{
			Slots:               nil,
			AvailabilityError:   "No available timeslots after enrichment",
			MaxAvailableDate:    maxDate,
			SubscriptionAllowed: provider.SubscriptionEnabled,
			SubscriptionModel:   provider.SubscriptionModel,
		}, nil
	}

	slots, err := BuildAvailableSlots(enriched, weekStart, weekEnd, now, provider.PaymentDetails.Currency)
	if err != nil {
		return AvailableSlotsResult{}, fmt.Errorf("failed to build available slots: %w", err)
	}

	// 5. Return everything, with subscription info straight from provider
	return AvailableSlotsResult{
		Slots:               slots,
		AvailabilityError:   "",
		MaxAvailableDate:    maxDate,
		SubscriptionAllowed: provider.SubscriptionEnabled,
		SubscriptionModel:   provider.SubscriptionModel,
	}, nil
}
