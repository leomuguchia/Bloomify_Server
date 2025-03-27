package booking

import (
	"fmt"
	"log"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"
)

func GetEnrichedTimeslots(repo providerRepo.ProviderRepository, providerID string) ([]models.TimeSlot, error) {
	log.Printf("DEBUG: GetEnrichedTimeslots: Calling repo for providerID: %s", providerID)
	prov, err := repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		fmt.Printf("INFO: Provider %s not found: %v. Returning empty timeslot list.\n", providerID, err)
		return []models.TimeSlot{}, nil
	}
	// Enrich timeslots: if ServiceCatalogue.Mode is set, assign it to each timeslot.
	if prov.ServiceCatalogue.ServiceType != "" {
		for i := range prov.TimeSlots {
			prov.TimeSlots[i].Mode = prov.ServiceCatalogue.Mode
		}
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
