package models

// BookingSession holds context between matching and final booking.
type BookingSession struct {
	ServicePlan      ServicePlan         `json:"servicePlan"`
	MatchedProviders []Provider          `json:"matchedProviders"`
	SelectedProvider string              `json:"selectedProviderId,omitempty"`
	Availability     []AvailableInterval `json:"availability,omitempty"`
}
