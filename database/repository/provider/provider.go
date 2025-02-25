package providerRepo

import "bloomify/models"

// ProviderSearchCriteria defines criteria for an advanced provider search.
type ProviderSearchCriteria struct {
	// Filter by service type (e.g., "Cleaning", "Laundry")
	ServiceType string
	// Free-text location filter (e.g., "New York")
	Location string
	// Minimum average rating required.
	MinRating float64
	// Minimum number of completed bookings.
	MinCompletedBookings int
	// For geospatial search: maximum distance (in km) from a given point.
	MaxDistanceKm float64
	// Center of the geospatial search.
	Latitude  float64
	Longitude float64
}

// ProviderRepository defines methods for provider data access.
type ProviderRepository interface {
	// GetByID retrieves a provider by its unique ID.
	GetByID(id string) (*models.Provider, error)
	// GetAll retrieves all providers.
	GetAll() ([]models.Provider, error)
	// GetByServiceType returns providers that offer a specific service.
	GetByServiceType(service string) ([]models.Provider, error)
	// Create inserts a new provider record.
	Create(provider *models.Provider) error
	// Update modifies an existing provider record.
	Update(provider *models.Provider) error
	// Delete removes a provider record by its ID.
	Delete(id string) error
	// AdvancedSearch performs an advanced search based on various criteria.
	AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error)
}
