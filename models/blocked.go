package models

import "time"

type Blocked struct {
	BlockID     string    `bson:"blockId" json:"blockId"`         // Unique identifier for the block
	ProviderID  string    `bson:"providerId" json:"providerId"`   // Provider whose slot is blocked
	Date        string    `bson:"date" json:"date"`               // Date (e.g., "2025-02-25")
	Start       int       `bson:"start" json:"start"`             // Start time in minutes from midnight
	End         int       `bson:"end" json:"end"`                 // End time in minutes from midnight
	Reason      string    `bson:"reason" json:"reason"`           // Reason for blocking (e.g., "capacity reached", "slot time expired")
	ServiceType string    `bson:"serviceType" json:"serviceType"` // Type of service (e.g., "Cleaning", "Daycare")
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`     // Timestamp when the block was created
}
