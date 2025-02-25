package models

import "time"

type Blocked struct {
	BlockID     string    `bson:"block_id" json:"block_id"`         // Unique identifier for the block
	ProviderID  string    `bson:"provider_id" json:"provider_id"`   // Provider whose slot is blocked
	Date        string    `bson:"date" json:"date"`                 // Date (e.g., "2025-02-25")
	Start       int       `bson:"start" json:"start"`               // Start time in minutes from midnight
	End         int       `bson:"end" json:"end"`                   // End time in minutes from midnight
	Reason      string    `bson:"reason" json:"reason"`             // Reason for blocking (e.g., "capacity reached", "slot time expired")
	ServiceType string    `bson:"service_type" json:"service_type"` // Type of service (e.g., "Cleaning", "Daycare")
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`     // Timestamp when the block was created
}
