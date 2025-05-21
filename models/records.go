// File: models/records.go
package models

import "time"

// carries record on of a past timeslot usage and experience
type HistoricalRecord struct {
	ID               string            `bson:"id" json:"id"`                   // Unique ID for the historical record
	ProviderID       string            `bson:"providerId" json:"providerId"`   // Owner provider
	TimeSlotID       string            `bson:"timeSlotId" json:"timeSlotId"`   // Original timeslot ID
	Date             string            `bson:"date" json:"date"`               // Date of the timeslot
	Bookings         []BookingSnapshot `bson:"bookings" json:"bookings"`       // All bookings with earnings & review snapshot
	TotalEarned      float64           `bson:"totalEarned" json:"totalEarned"` // Sum of earnings from all users
	Capacity         int               `bson:"capacity" json:"capacity"`       // Total capacity of the timeslot
	ServiceCatalogue ServiceCatalogue  `bson:"serviceCatalogue" json:"serviceCatalogue"`
	CreatedAt        time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time         `bson:"updatedAt" json:"updatedAt"`
}

type BookingSnapshot struct {
	UserID    string  `bson:"userId" json:"userId"`                     // User who booked
	BookingID string  `bson:"bookingId" json:"bookingId"`               // Link to actual booking
	Earned    float64 `bson:"earned" json:"earned"`                     // What the provider earned from this user
	Review    Review  `bson:"review,omitempty" json:"review,omitempty"` // Snapshot of the review
}

type Review struct {
	Rating  float64 `bson:"rating" json:"rating"`   // Expected value between 1 and 5.
	Comment string  `bson:"comment" json:"comment"` // Customer's feedback.
}
