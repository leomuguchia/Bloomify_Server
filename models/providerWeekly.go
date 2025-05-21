package models

import "time"

// WeeklyTemplate holds one week’s “full‐day” setup.
type WeeklyTemplate struct {
	AnchorDate string         `json:"anchorDate" binding:"required"` // e.g. "2025-06-02"
	ActiveDays []time.Weekday `json:"activeDays" binding:"required"` // weekdays to replicate onto
	BaseSlots  []TimeSlot     `json:"baseSlots"  binding:"required"` // slots provider created for AnchorDate
}

type SetupTimeslotsRequest struct {
	Weeks []WeeklyTemplate `json:"weeks" binding:"required,min=2,max=6"`
}
