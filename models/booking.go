// models/booking.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// Booking represents a booking record.
type Booking struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ProviderID  string         `json:"provider_id"`
	UserID      uint           `json:"user_id"`
	Date        string         `json:"date"`         // YYYY-MM-DD
	StartMinute int            `json:"start_minute"` // Minutes from midnight
	Duration    int            `json:"duration"`     // Duration in minutes
	Units       int            `json:"units"`        // Number of capacity units booked
	TotalPrice  float64        `json:"total_price"`  // Calculated total price
	Status      string         `json:"status"`       // "Confirmed", "Pending", "Cancelled"
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
