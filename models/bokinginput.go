package models

// BookingRequestInput holds provider selection details.
type BookingRequestInput struct {
	ProviderID string `json:"providerID"` // The selected provider's ID.
	Date       string `json:"date"`       // Booking date (YYYY-MM-DD).
	Duration   int    `json:"duration"`   // Duration in minutes.
	Units      int    `json:"units"`      // Number of capacity units requested.
}
