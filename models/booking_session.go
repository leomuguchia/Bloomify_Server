package models

// BookingSession holds context between matching and final booking.
type BookingSession struct {
	SessionID        string          `json:"sessionId"`
	ServicePlan      ServicePlan     `json:"servicePlan"`
	MatchedProviders []Provider      `json:"matchedProviders"`
	SelectedProvider string          `json:"selectedProviderId,omitempty"`
	Availability     []AvailableSlot `json:"availability,omitempty"`
	UserID           string          `json:"user_id"`        // The user making the booking.
	PaymentMethod    string          `json:"payment_method"` // Should be "inApp".
}

type BookingResponse struct {
	SessionID    string          `json:"sessionID,omitempty"`
	Providers    []Provider      `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}
