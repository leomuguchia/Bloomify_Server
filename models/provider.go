package models

import (
	"time"

	"gorm.io/gorm"
)

// Provider represents a service provider.
type Provider struct {
	ID                string         `gorm:"primaryKey" json:"id"`
	Name              string         `json:"name"`
	Email             string         `json:"email"`
	PhoneNumber       string         `json:"phone_number"`
	WorkingStart      int            `json:"working_start"` // Minutes from midnight
	WorkingEnd        int            `json:"working_end"`   // Minutes from midnight
	Capacity          int            `json:"capacity"`      // Number of resource units available concurrently
	ServiceType       string         `json:"service_type"`  // e.g., "Cleaning", "Laundry", etc.
	PricingModel      string         `json:"pricing_model"` // e.g., "Hourly", "PerUnit", "FlatRate"
	BaseRate          float64        `json:"base_rate"`
	Location          string         `json:"location"`           // Human-readable location (e.g., city)
	Latitude          float64        `json:"latitude"`           // Geographic latitude
	Longitude         float64        `json:"longitude"`          // Geographic longitude
	Rating            float64        `json:"rating"`             // Historical average rating (e.g., 0.0 to 5.0)
	CompletedBookings int            `json:"completed_bookings"` // Count of completed bookings
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
