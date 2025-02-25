package models

// ServicePlan defines the parameters for a client's service request.
type ServicePlan struct {
	Service    string  `json:"service"`     // e.g., "Laundry", "Cleaning"
	BookingFor string  `json:"booking_for"` // e.g., "Myself", "Family"
	Priority   bool    `json:"priority"`    // True if the request is urgent/priority
	Location   string  `json:"location"`    // e.g., "New York"
	Latitude   float64 `json:"latitude"`    // Requester's latitude
	Longitude  float64 `json:"longitude"`   // Requester's longitude
	Date       string  `json:"date"`        // Date in "YYYY-MM-DD" format
}
