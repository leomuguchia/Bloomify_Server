package booking

import (
	"fmt"
	"time"

	providerRepo "bloomify/database/repository/provider"
	schedulerRepo "bloomify/database/repository/scheduler"
	"bloomify/models"
	"bloomify/utils"

	"go.uber.org/zap"
)

// DefaultSchedulingEngine is our production-grade scheduler.
type DefaultSchedulingEngine struct {
	Repo           schedulerRepo.SchedulerRepository
	PaymentHandler PaymentHandler
	ProviderRepo   providerRepo.ProviderRepository
}

type AvailableSlotsResult struct {
	Slots               []models.AvailableSlot
	Mapping             map[string]models.TimeSlot
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
	maxDate, err := se.Repo.GetMaxAvailableDate(provider.ID)
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
		daySlots, err := se.Repo.GetAvailableTimeSlots(provider.ID, dateStr)
		if err != nil {
			logger.Error("error fetching timeslots", zap.String("date", dateStr), zap.Error(err))
			continue
		}
		raw = append(raw, daySlots...)
	}
	if len(raw) == 0 {
		return AvailableSlotsResult{
			Slots:             nil,
			Mapping:           map[string]models.TimeSlot{},
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
			Mapping:             map[string]models.TimeSlot{},
			AvailabilityError:   "No available timeslots after enrichment",
			MaxAvailableDate:    maxDate,
			SubscriptionAllowed: provider.SubscriptionEnabled,
			SubscriptionModel:   provider.SubscriptionModel,
		}, nil
	}

	slots, mapping, err := BuildAvailableSlots(enriched, weekStart, weekEnd, now)
	if err != nil {
		return AvailableSlotsResult{}, fmt.Errorf("failed to build available slots: %w", err)
	}

	// 5. Return everything, with subscription info straight from provider
	return AvailableSlotsResult{
		Slots:               slots,
		Mapping:             mapping,
		AvailabilityError:   "",
		MaxAvailableDate:    maxDate,
		SubscriptionAllowed: provider.SubscriptionEnabled,
		SubscriptionModel:   provider.SubscriptionModel,
	}, nil
}
