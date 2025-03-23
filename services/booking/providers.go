package booking

import (
	"fmt"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"
)

// GetEnrichedTimeslots retrieves and enriches the provider's timeslots with the mode from the service catalogue.
func GetEnrichedTimeslots(repo providerRepo.ProviderRepository, providerID string) ([]models.TimeSlot, error) {
	prov, err := repo.GetByID(providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}
	// Enrich timeslots: attach the delivery mode from the ServiceCatalogue.
	for i := range prov.TimeSlots {
		prov.TimeSlots[i].Mode = prov.ServiceCatalogue.Mode
		// Leave CustomOptionKey intact for pricing lookup.
	}
	return prov.TimeSlots, nil
}

// getCapacityMessage computes a friendly capacity message based on the remaining capacity.
func getCapacityMessage(ts models.TimeSlot) string {
	switch ts.SlotModel {
	case "urgency":
		if ts.Urgency == nil {
			return ""
		}
		normalCapacity := ts.Capacity - ts.Urgency.ReservedPriority
		remaining := normalCapacity - ts.BookedUnitsStandard
		if normalCapacity > 0 && float64(remaining)/float64(normalCapacity) < 0.3 {
			return fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
		}
	case "earlybird":
		remaining := ts.Capacity - ts.BookedUnitsStandard
		if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
			return fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
		}
	default: // flatrate or standard.
		remaining := ts.Capacity - ts.BookedUnitsStandard
		if ts.Capacity > 0 && float64(remaining)/float64(ts.Capacity) < 0.3 {
			return fmt.Sprintf("Only %d %s remaining", remaining, ts.UnitType)
		}
	}
	return ""
}
