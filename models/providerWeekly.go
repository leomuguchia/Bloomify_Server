package models

type WeekSchedule struct {
	StartDate  string     `json:"startDate" binding:"required"` // "2025-06-23"
	BaseSlots  []TimeSlot `json:"baseSlots" binding:"required"`
	ActiveDays []string   `json:"activeDays" binding:"required"` // e.g. ["Mon", "Tue"]
}

type SetupTimeslotsRequest struct {
	Weeks []WeekSchedule `json:"weeks" binding:"required"`
}
