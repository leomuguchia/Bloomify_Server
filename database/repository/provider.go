package repository

import (
	"strings"

	"bloomify/database"
	"bloomify/models"
)

// ProviderRepository defines the interface for provider data access.
type ProviderRepository interface {
	GetByID(id string) (*models.Provider, error)
	GetAll() ([]models.Provider, error)
	GetByServiceType(serviceType string) ([]models.Provider, error)
	Create(provider *models.Provider) error
	Update(provider *models.Provider) error
	Delete(id string) error
}

// GormProviderRepo implements ProviderRepository using GORM.
type GormProviderRepo struct{}

// GetByID retrieves a provider by its ID.
func (repo *GormProviderRepo) GetByID(id string) (*models.Provider, error) {
	var provider models.Provider
	err := database.DB.First(&provider, "id = ?", id).Error
	return &provider, err
}

// GetAll retrieves all providers.
func (repo *GormProviderRepo) GetAll() ([]models.Provider, error) {
	var providers []models.Provider
	err := database.DB.Find(&providers).Error
	return providers, err
}

// GetByServiceType retrieves all providers that match a given service type.
func (repo *GormProviderRepo) GetByServiceType(serviceType string) ([]models.Provider, error) {
	var providers []models.Provider
	err := database.DB.Where("LOWER(service_type) = ?", strings.ToLower(serviceType)).Find(&providers).Error
	return providers, err
}

// Create inserts a new provider record.
func (repo *GormProviderRepo) Create(provider *models.Provider) error {
	return database.DB.Create(provider).Error
}

// Update saves the updated provider record.
func (repo *GormProviderRepo) Update(provider *models.Provider) error {
	return database.DB.Save(provider).Error
}

// Delete removes the provider record by ID.
func (repo *GormProviderRepo) Delete(id string) error {
	return database.DB.Delete(&models.Provider{}, "id = ?", id).Error
}

// In bloomify/database/repository/provider.go
func NewGormProviderRepo() ProviderRepository {
	return &GormProviderRepo{}
}
