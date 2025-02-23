// models/service_type.go
package models

// ServiceType represents a type of service offered on the platform.
type ServiceType struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	Name            string `json:"name"`             // e.g., "Cleaning", "Chauffeur"
	DefaultDuration int    `json:"default_duration"` // in minutes
	PricingUnit     string `json:"pricing_unit"`     // e.g., "Hour", "Load"
	Description     string `json:"description"`
}
