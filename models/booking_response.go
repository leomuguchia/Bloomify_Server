// models/booking_response.go
package models

// BookingResponse is returned by the unified booking engine.
// It may either contain a session context (when the booking process is in progress)
// or a fully confirmed booking.
type BookingResponse struct {
	// SessionID is provided when the booking process is not yet finalized.
	SessionID string `json:"sessionID,omitempty"`
	// Providers is provided with the initial matching results.
	Providers []Provider `json:"providers,omitempty"`
	// Availability is provided when a provider has been selected.
	Availability []AvailableInterval `json:"availability,omitempty"`
	// Booking is set when the booking is confirmed.
	Booking *Booking `json:"booking,omitempty"`
}
