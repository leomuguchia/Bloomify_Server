package models

// TimeSlot represents a provider's pre-defined booking window.
type TimeSlot struct {
	ID        string `bson:"id"`
	Start     int    `bson:"start" json:"start"`           // minutes from midnight (e.g., 420 for 7:00 AM)
	End       int    `bson:"end" json:"end"`               // minutes from midnight (e.g., 780 for 1:00 PM)
	Capacity  int    `bson:"capacity" json:"capacity"`     // total units for the slot (e.g., 30 kids)
	SlotModel string `bson:"slot_model" json:"slot_model"` // SlotModel indicates which pricing model is used: "earlybird", "urgency", or "flatrate"
	UnitType  string `bson:"unit_type" json:"unit_type"`   // NEW: e.g., "child", "kg", "hour"
	Date      string `bson:"date,omitempty" json:"date"`   // new field: e.g., "2025-02-25"

	// Depending on SlotModel, only one of these will be non-nil.
	EarlyBird *EarlyBirdSlotData `bson:"earlybird,omitempty" json:"earlybird,omitempty"`
	Urgency   *UrgencySlotData   `bson:"urgency,omitempty" json:"urgency,omitempty"`
	Flatrate  *FlatrateSlotData  `bson:"flatrate,omitempty" json:"flatrate,omitempty"`

	BookedUnitsStandard int `bson:"booked_units_standard,omitempty" json:"booked_units_standard,omitempty"`
	BookedUnitsPriority int `bson:"booked_units_priority,omitempty" json:"booked_units_priority,omitempty"`
	Version             int `bson:"version" json:"version"`
}

// AvailableSlot represents a time slot with its remaining capacity, unit type, pricing, and an optional message.
type AvailableSlot struct {
	Start                     int     `bson:"start" json:"start"`
	End                       int     `bson:"end" json:"end"`
	UnitType                  string  `bson:"unit_type" json:"unit_type"` // e.g., "child", "kg", "hour"
	RegularCapacityRemaining  int     `bson:"regular_capacity_remaining" json:"regular_capacity_remaining"`
	PriorityCapacityRemaining int     `bson:"priority_capacity_remaining,omitempty" json:"priority_capacity_remaining,omitempty"`
	RegularPricePerUnit       float64 `bson:"regular_price_per_unit,omitempty" json:"regular_price_per_unit,omitempty"`
	PriorityPricePerUnit      float64 `bson:"priority_price_per_unit,omitempty" json:"priority_price_per_unit,omitempty"`
	Message                   string  `bson:"message,omitempty" json:"message,omitempty"`
	Date                      string  `bson:"date,omitempty" json:"date"`
}

type EarlyBirdSlotData struct {
	BasePrice             float64 `bson:"base_price" json:"base_price"`                             // base price per unit.
	EarlyBirdDiscountRate float64 `bson:"early_bird_discount_rate" json:"early_bird_discount_rate"` // applied for the early tier (e.g., 0.25 for 25% discount).
	LateSurchargeRate     float64 `bson:"late_surcharge_rate" json:"late_surcharge_rate"`           // applied for the late tier (e.g., 0.25 for 25% surcharge).
}

type UrgencySlotData struct {
	BasePrice             float64 `bson:"base_price" json:"base_price"`                                   // base price per unit.
	PrioritySurchargeRate float64 `bson:"priority_surcharge_rate" json:"priority_surcharge_rate"`         // applied for bookings in the priority bucket (e.g., 0.50 for 50% surcharge).
	ReservedPriority      int     `bson:"reserved_priority,omitempty" json:"reserved_priority,omitempty"` // capacity reserved for urgent (priority) bookings.
	PriorityActive        bool    `bson:"priority_active" json:"priority_active"`
}

type FlatrateSlotData struct {
	BasePrice float64 `bson:"base_price" json:"base_price"`
}

// ProviderTimeslotDTO represents a minimal view for timeslot setup.
type ProviderTimeslotDTO struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	TimeSlots []TimeSlot `json:"time_slots"`
}

// SetupTimeslotsRequest defines the payload for setting up timeslots.
type SetupTimeslotsRequest struct {
	TimeSlots []TimeSlot `json:"time_slots" binding:"required"`
}
