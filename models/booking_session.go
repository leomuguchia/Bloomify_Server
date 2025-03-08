package models

type BookingSession struct {
	SessionID        string          `json:"sessionId"`
	ServicePlan      ServicePlan     `json:"servicePlan"`
	MatchedProviders []ProviderDTO   `json:"matchedProviders"`
	SelectedProvider string          `json:"selectedProviderId,omitempty"`
	Availability     []AvailableSlot `json:"availability,omitempty"`
	UserID           string          `json:"user_id"`
	PaymentMethod    string          `json:"payment_method"`
}

type BookingResponse struct {
	SessionID    string          `json:"sessionID,omitempty"`
	Providers    []ProviderDTO   `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}
