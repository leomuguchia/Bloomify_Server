package models

// BookingSession holds context between matching and final booking.
type BookingSession struct {
	SessionID        string          `json:"sessionId"`
	ServicePlan      ServicePlan     `json:"servicePlan"`
	MatchedProviders []Provider      `json:"matchedProviders"`
	SelectedProvider string          `json:"selectedProviderId,omitempty"`
	Availability     []AvailableSlot `json:"availability,omitempty"`
}

type BookingResponse struct {
	SessionID    string          `json:"sessionID,omitempty"`
	Providers    []Provider      `json:"providers,omitempty"`
	Availability []AvailableSlot `json:"availability,omitempty"`
	Booking      *Booking        `json:"booking,omitempty"`
}

// BookingConfirmationSession represents the context needed during confirmation.
type BookingConfirmationSession struct {
	SelectedProvider string          `json:"selected_provider"` // Provider ID chosen by the user.
	UserID           string          `json:"user_id"`           // The user making the booking.
	ServicePlan      ServicePlan     `json:"service_plan"`      // Service plan details.
	Availability     []AvailableSlot `json:"availability"`      // List of available intervals.
	PaymentMethod    string          `json:"payment_method"`    // Should be "inApp".
}
