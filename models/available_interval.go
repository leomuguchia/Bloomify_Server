package models

// AvailableInterval represents a continuous time block available for booking.
type AvailableInterval struct {
	Start int    `json:"start"` // Minutes from midnight
	End   int    `json:"end"`   // Minutes from midnight
	Label string `json:"label"` // e.g., "9:00 AM - 10:30 AM"
}
