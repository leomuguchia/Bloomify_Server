package models

// TimeSlot represents a provider's pre-defined booking window.
type TimeSlot struct {
	ID                  string             `bson:"id" json:"id"`
	Start               int                `bson:"start" json:"start"`                             // minutes from midnight (e.g., 420 for 7:00 AM)
	End                 int                `bson:"end" json:"end"`                                 // minutes from midnight (e.g., 780 for 1:00 PM)
	Capacity            int                `bson:"capacity" json:"capacity"`                       // total units for the slot (e.g., 30 kids)
	SlotModel           string             `bson:"slotModel" json:"slotModel"`                     // "earlybird", "urgency", or "flatrate"
	UnitType            string             `bson:"unitType" json:"unitType"`                       // e.g., "child", "kg", "hour"
	Date                string             `bson:"date,omitempty" json:"date"`                     // e.g., "2025-02-25"
	EarlyBird           *EarlyBirdSlotData `bson:"earlyBird,omitempty" json:"earlyBird,omitempty"` // non-nil when SlotModel is "earlybird"
	Urgency             *UrgencySlotData   `bson:"urgency,omitempty" json:"urgency,omitempty"`     // non-nil when SlotModel is "urgency"
	Flatrate            *FlatrateSlotData  `bson:"flatRate,omitempty" json:"flatRate,omitempty"`   // non-nil when SlotModel is "flatrate"
	BookedUnitsStandard int                `bson:"bookedUnitsStandard,omitempty" json:"bookedUnitsStandard,omitempty"`
	BookedUnitsPriority int                `bson:"bookedUnitsPriority,omitempty" json:"bookedUnitsPriority,omitempty"`
	Version             int                `bson:"version" json:"version"`
	Catalogue           ServiceCatalogue   `bson:"catalogue,omitempty" json:"catalogue,omitzero"`
	Blocked             bool               `bson:"blocked" json:"blocked"`
	BlockReason         string             `bson:"blockReason,omitempty" json:"blockReason,omitempty"`
	BookingIDs          []string           `bson:"bookingIds,omitempty" json:"bookingIds,omitempty"`
}

// AvailableSlotResponse represents the detailed timeslot information including the user’s selected custom option and units.
type AvailableSlotResponse struct {
	ID           string        `json:"id"`
	Start        int           `json:"start"`
	End          int           `json:"end"`
	UnitType     string        `json:"unitType"`
	Date         string        `json:"date"`
	Units        int           `json:"units"`
	CustomOption *CustomOption `json:"customOption,omitempty"`
}

type CustomOption struct {
	Option string  `json:"option"`
	Price  float64 `json:"price"`
}

type AvailableSlot struct {
	ID                        string             `json:"id"`
	Start                     int                `json:"start"`
	End                       int                `json:"end"`
	UnitType                  string             `json:"unitType"`
	RegularCapacityRemaining  int                `json:"regularCapacityRemaining"`
	PriorityCapacityRemaining int                `json:"priorityCapacityRemaining,omitempty"`
	RegularPricePerUnit       float64            `json:"regularPricePerUnit,omitempty"`
	PriorityPricePerUnit      float64            `json:"priorityPricePerUnit,omitempty"`
	Message                   string             `json:"message,omitempty"`
	Date                      string             `json:"date"`
	CustomOptionKey           string             `json:"customOptionKey,omitempty"`
	Catalogue                 ServiceCatalogue   `bson:"catalogue,omitempty" json:"catalogue,omitzero"`
	OptionPricing             map[string]float64 `json:"optionPricing,omitempty"`
	CustomOption              *CustomOption      `json:"customOption,omitempty"`
}

type EarlyBirdSlotData struct {
	BasePrice             float64 `bson:"basePrice" json:"basePrice"`                         // base price per unit
	EarlyBirdDiscountRate float64 `bson:"earlyBirdDiscountRate" json:"earlyBirdDiscountRate"` // e.g., 0.25 for 25% discount
	LateSurchargeRate     float64 `bson:"lateSurchargeRate" json:"lateSurchargeRate"`         // e.g., 0.25 for 25% surcharge
}

type UrgencySlotData struct {
	BasePrice             float64 `bson:"basePrice" json:"basePrice"`                                   // base price per unit
	PrioritySurchargeRate float64 `bson:"prioritySurchargeRate" json:"prioritySurchargeRate"`           // e.g., 0.50 for 50% surcharge
	ReservedPriority      int     `bson:"reservedPriority,omitempty" json:"reservedPriority,omitempty"` // capacity reserved for urgent bookings
	PriorityActive        bool    `bson:"priorityActive" json:"priorityActive"`
}

type FlatrateSlotData struct {
	BasePrice float64 `bson:"basePrice" json:"basePrice"`
}

// ProviderTimeslotDTO represents a minimal view for timeslot setup.
type ProviderTimeslotDTO struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	TimeSlots []TimeSlot `json:"timeSlots"`
}

// SetupTimeslotsRequest defines the payload for setting up timeslots.
type SetupTimeslotsRequest struct {
	TimeSlots []TimeSlot `json:"timeSlots" binding:"required"`
}
