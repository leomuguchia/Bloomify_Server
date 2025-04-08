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
	PaymentHandler PaymentProcessor
	ProviderRepo   providerRepo.ProviderRepository
}

type AvailableSlotsResult struct {
	Slots             []models.AvailableSlot
	Mapping           map[string]models.TimeSlot
	AvailabilityError string `json:"availabilityError,omitempty"`
	MaxAvailableDate  string `json:"maxAvailableDate,omitempty"`
}

func (se *DefaultSchedulingEngine) GetWeeklyAvailableSlots(provider models.Provider, weekIndex int) (AvailableSlotsResult, error) {
	logger := utils.GetLogger()
	now := time.Now()

	// Retrieve the max available date from today onward.
	maxDate, err := se.Repo.GetMaxAvailableDate(provider.ID)
	if err != nil {
		logger.Error("GetWeeklyAvailableSlots: error fetching max available date",
			zap.String("providerID", provider.ID), zap.Error(err))
	}

	logger.Debug("GetMaxAvailableDate: aggregation results", zap.Any("results", maxDate))

	// Define week boundaries. Week 0 starts "today" (adjust if necessary).
	weekZeroStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := weekZeroStart.AddDate(0, 0, weekIndex*7)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Create a slice of date strings for each day in the week.
	var weekDates []string
	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		weekDates = append(weekDates, d.Format("2006-01-02"))
	}

	// Fetch raw timeslots for each day.
	var rawTimeslots []models.TimeSlot
	for _, dateStr := range weekDates {
		daySlots, err := se.Repo.GetAvailableTimeSlots(provider.ID, dateStr)
		if err != nil {
			logger.Error("GetWeeklyAvailableSlots: error fetching timeslots",
				zap.String("providerID", provider.ID),
				zap.String("date", dateStr),
				zap.Error(err))
			// Continue with next day.
			continue
		}
		rawTimeslots = append(rawTimeslots, daySlots...)
	}

	// If no raw timeslots found, return with an appropriate error message.
	if len(rawTimeslots) == 0 {
		return AvailableSlotsResult{
			Slots:             []models.AvailableSlot{},
			Mapping:           map[string]models.TimeSlot{},
			AvailabilityError: "No schedule available for the selected provider",
			MaxAvailableDate:  maxDate,
		}, nil
	}

	// Enrich the raw timeslots.
	enrichedTimeslots := EnrichTimeslots(rawTimeslots, provider.ServiceCatalogue, logger)
	if len(enrichedTimeslots) == 0 {
		return AvailableSlotsResult{
			Slots:             []models.AvailableSlot{},
			Mapping:           map[string]models.TimeSlot{},
			AvailabilityError: "No available timeslots after enrichment",
			MaxAvailableDate:  maxDate,
		}, nil
	}

	// Build the final available slots and mapping.
	slots, mapping, err := BuildAvailableSlots(enrichedTimeslots, provider.ServiceCatalogue, weekStart, weekEnd, now)
	if err != nil {
		return AvailableSlotsResult{}, fmt.Errorf("failed to build available slots: %w", err)
	}

	return AvailableSlotsResult{
		Slots:             slots,
		Mapping:           mapping,
		AvailabilityError: "",
		MaxAvailableDate:  maxDate,
	}, nil
}
