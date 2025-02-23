// database/repository/blocked.go
package repository

import (
	"bloomify/database"
	"bloomify/models"
)

// BlockedRepository defines methods to interact with blocked intervals.
type BlockedRepository interface {
	GetByProviderAndDate(providerID, date string) ([]models.Blocked, error)
	Create(block *models.Blocked) error
	Delete(id uint) error
}

// GormBlockedRepo implements BlockedRepository using GORM.
type GormBlockedRepo struct{}

func (repo *GormBlockedRepo) GetByProviderAndDate(providerID, date string) ([]models.Blocked, error) {
	var blocks []models.Blocked
	err := database.DB.Where("provider_id = ? AND date = ?", providerID, date).Find(&blocks).Error
	return blocks, err
}

func (repo *GormBlockedRepo) Create(block *models.Blocked) error {
	return database.DB.Create(block).Error
}

func (repo *GormBlockedRepo) Delete(id uint) error {
	return database.DB.Delete(&models.Blocked{}, "id = ?", id).Error
}
