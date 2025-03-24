package booking

import (
	"bloomify/models"
	"fmt"
)

func validateServicePlan(plan models.ServicePlan) error {
	if plan.ServiceType == "" {
		return fmt.Errorf("serviceType is required")
	}
	if plan.BookingFor == "" {
		return fmt.Errorf("bookingFor is required")
	}
	if plan.Mode == "" {
		return fmt.Errorf("serviceMode is required")
	}
	if plan.LocationGeo.Type != "Point" {
		return fmt.Errorf("locationGeo.type must be 'Point'")
	}
	// Omit explicit nil check; len(nil) returns 0.
	if len(plan.LocationGeo.Coordinates) < 2 {
		return fmt.Errorf("locationGeo.coordinates must be an array of at least two numbers")
	}
	if plan.Date == "" {
		return fmt.Errorf("date is required")
	}
	if plan.Units <= 0 {
		return fmt.Errorf("units must be a positive integer")
	}
	if plan.UnitType == "" {
		return fmt.Errorf("unitType is required")
	}
	return nil
}
