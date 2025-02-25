package models

import "time"

// Booking represents a confirmed booking record.
type Booking struct {
	ID         string    `bson:"id" json:"id"`                                 // Unique booking identifier (e.g., UUID)
	ProviderID string    `bson:"provider_id" json:"provider_id"`               // Provider who was booked
	UserID     string    `bson:"user_id" json:"user_id"`                       // User who made the booking
	Date       string    `bson:"date" json:"date"`                             // Booking date in "YYYY-MM-DD" format
	Units      int       `bson:"units" json:"units"`                           // Number of capacity units booked (e.g., 1, 2, etc.)
	UnitType   string    `bson:"unit_type" json:"unit_type"`                   // Measurement unit (e.g., "child", "kg", "hour")
	TotalPrice float64   `bson:"total_price" json:"total_price"`               // Calculated total price
	Status     string    `bson:"status" json:"status"`                         // e.g., "Confirmed", "Pending"
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`                 // Timestamp when booking was created
	Start      int       `bson:"start" json:"start"`                           // Booking start time (minutes from midnight)
	End        int       `bson:"end" json:"end"`                               // Booking end time (minutes from midnight)
	Priority   bool      `bson:"priority,omitempty" json:"priority,omitempty"` // Indicates if the booking is under the urgency (priority) bucket
}
