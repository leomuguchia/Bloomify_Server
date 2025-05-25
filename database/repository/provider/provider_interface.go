package providerRepo

import (
	"bloomify/database"
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ProviderSearchCriteria defines criteria for an advanced provider search.
type ProviderSearchCriteria struct {
	ServiceType   string
	Location      string
	MaxDistanceKm float64
	LocationGeo   models.GeoPoint
	Mode          string
	CustomOption  string
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
	// UpdateWithDocument patches a provider document with the specified update document.
	UpdateWithDocument(id string, updateDoc bson.M) error
	// IsProviderAvailable checks if a provider with the given basic registration details already exists.
	IsProviderAvailable(basicReq models.ProviderBasicRegistrationData) (bool, error)
	FetchTopProviders(ctx context.Context, page, limit int) ([]models.Provider, error)
}

// MongoProviderRepo implements ProviderRepository using MongoDB.
type MongoProviderRepo struct {
	coll *mongo.Collection
}

// NewMongoProviderRepo creates a new instance of ProviderRepository using MongoDB.
func NewMongoProviderRepo() ProviderRepository {
	coll := database.MongoClient.Database("bloomify").Collection("providers")
	repo := &MongoProviderRepo{coll: coll}

	if err := repo.ensureIndexes(); err != nil {
		fmt.Printf("failed to create indexes: %v\n", err)
	}
	return repo
}

// newContext creates a context with the given timeout.
func newContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
