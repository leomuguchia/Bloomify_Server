package providerRepo

import (
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// ProviderSearchCriteria defines criteria for an advanced provider search.
type ProviderSearchCriteria struct {
	ServiceType          string
	Location             string
	MinRating            float64
	MinCompletedBookings int
	MaxDistanceKm        float64
	LocationGeo          models.GeoPoint
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
	// GetByTokenHash retrieves a provider whose token_hash matches the provided hash.
	GetByTokenHash(tokenHash string) (*models.Provider, error)
	// UpdateWithDocument patches a provider document with the specified update document.
	UpdateWithDocument(id string, updateDoc bson.M) error
}
