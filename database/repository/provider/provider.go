package providerRepo

import (
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

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
	// GetByEmail retrieves a provider by its email address.
	GetByEmail(email string) (*models.Provider, error)
	// AdvancedSearch performs an advanced search based on various criteria.
	AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error)
	// GetByIDWithProjection retrieves a provider by its unique ID with a projection.
	GetByIDWithProjection(id string, projection bson.M) (*models.Provider, error)
	// GetByEmailWithProjection retrieves a provider by its email with a projection.
	GetByEmailWithProjection(email string, projection bson.M) (*models.Provider, error)
	// GetAllWithProjection retrieves all providers with an optional projection.
	GetAllWithProjection(projection bson.M) ([]models.Provider, error)
	// GetByServiceTypeWithProjection retrieves providers by service type with a projection.
	GetByServiceTypeWithProjection(service string, projection bson.M) ([]models.Provider, error)
}
