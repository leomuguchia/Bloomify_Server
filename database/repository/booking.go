// database/repository/booking.go
package repository

import (
	"bloomify/database"
	"bloomify/models"
)

// BookingRepository defines the interface for booking data access.
type BookingRepository interface {
	GetByID(id uint) (*models.Booking, error)
	Create(booking *models.Booking) error
	Update(booking *models.Booking) error
	// Additional methods like listing bookings by provider/date can be added.
}

// GormBookingRepo implements BookingRepository using GORM.
type GormBookingRepo struct{}

func (repo *GormBookingRepo) GetByID(id uint) (*models.Booking, error) {
	var booking models.Booking
	err := database.DB.First(&booking, "id = ?", id).Error
	return &booking, err
}

func (repo *GormBookingRepo) Create(booking *models.Booking) error {
	return database.DB.Create(booking).Error
}

func (repo *GormBookingRepo) Update(booking *models.Booking) error {
	return database.DB.Save(booking).Error
}
