package providerRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/database"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// GetByTokenHash retrieves a provider by its tokenHash using a projection.
func (r *MongoProviderRepo) GetByTokenHash(tokenHash string) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne().SetProjection(bson.M{"tokenHash": 1, "id": 1})
	var result struct {
		ID        string `bson:"id"`
		TokenHash string `bson:"tokenHash"`
	}
	if err := r.coll.FindOne(ctx, bson.M{"tokenHash": tokenHash}, opts).Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve provider by token hash: %w", err)
	}

	return r.GetByID(result.ID)
}

// GetByIDWithProjection retrieves a provider by its unique ID using a projection.
// Pass nil for projection if you want the full document with sensitive fields omitted by default.
func (r *MongoProviderRepo) GetByIDWithProjection(id string, projection bson.M) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}

	opts := options.FindOne().SetProjection(proj)
	var provider models.Provider
	if err := r.coll.FindOne(ctx, bson.M{"id": id}, opts).Decode(&provider); err != nil {
		return nil, fmt.Errorf("failed to fetch provider with id %s: %w", id, err)
	}

	if provider.Devices == nil {
		provider.Devices = []models.Device{}
	}

	return &provider, nil
}

// GetByEmailWithProjection retrieves a provider by its email using a projection.
// Pass nil for projection to retrieve the full document with sensitive fields omitted by default.
func (r *MongoProviderRepo) GetByEmailWithProjection(email string, projection bson.M) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}

	opts := options.FindOne().SetProjection(proj)
	var provider models.Provider
	if err := r.coll.FindOne(ctx, bson.M{"profile.email": email}, opts).Decode(&provider); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch provider with email %s: %w", email, err)
	}

	if provider.Devices == nil {
		provider.Devices = []models.Device{}
	}

	return &provider, nil
}

// GetAllWithProjection retrieves all providers with an optional projection.
// If projection is nil, a default projection is applied.
func (r *MongoProviderRepo) GetAllWithProjection(projection bson.M) ([]models.Provider, error) {
	ctx, cancel := newContext(10 * time.Second)
	defer cancel()

	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}

	opts := options.Find().SetProjection(proj)
	// Use an empty filter to match all documents.
	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve providers: %w", err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	for cursor.Next(ctx) {
		var p models.Provider
		if err := cursor.Decode(&p); err != nil {
			return nil, fmt.Errorf("failed to decode provider: %w", err)
		}
		if p.Devices == nil {
			p.Devices = []models.Device{}
		}
		providers = append(providers, p)
	}
	return providers, nil
}

// GetByServiceTypeWithProjection retrieves providers matching the given service type with a projection.
func (r *MongoProviderRepo) GetByServiceTypeWithProjection(service string, projection bson.M) ([]models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Adjust filter to refer to serviceCatalogue.serviceType.
	filter := bson.M{"serviceCatalogue.serviceType": bson.M{"$regex": service, "$options": "i"}}
	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}

	opts := options.Find().SetProjection(proj)
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find providers for service %s: %w", service, err)
	}
	defer cursor.Close(ctx)

	var providers []models.Provider
	for cursor.Next(ctx) {
		var p models.Provider
		if err := cursor.Decode(&p); err != nil {
			return nil, fmt.Errorf("failed to decode provider: %w", err)
		}
		if p.Devices == nil {
			p.Devices = []models.Device{}
		}
		providers = append(providers, p)
	}
	return providers, nil
}

// --- Exported Query Methods ---

// GetByID retrieves a provider by its unique ID (full document).
func (r *MongoProviderRepo) GetByID(id string) (*models.Provider, error) {
	return r.GetByIDWithProjection(id, nil)
}

// GetByEmail retrieves a provider by its email address (full document).
func (r *MongoProviderRepo) GetByEmail(email string) (*models.Provider, error) {
	return r.GetByEmailWithProjection(email, nil)
}

// GetAll retrieves all providers (full documents).
func (r *MongoProviderRepo) GetAll() ([]models.Provider, error) {
	return r.GetAllWithProjection(nil)
}

// GetByServiceType returns providers that offer a specific service (full documents).
func (r *MongoProviderRepo) GetByServiceType(service string) ([]models.Provider, error) {
	return r.GetByServiceTypeWithProjection(service, nil)
}

// IsProviderAvailable checks if a provider with the given email or username already exists.
func (r *MongoProviderRepo) IsProviderAvailable(basicReq models.ProviderBasicRegistrationData) (bool, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Adjusted filter: check within profile for email and providerName.
	filter := bson.M{
		"$or": []bson.M{
			{"profile.email": basicReq.Email},
			{"profile.providerName": basicReq.Username},
		},
	}

	var provider models.Provider
	err := r.coll.FindOne(ctx, filter).Decode(&provider)
	if err != nil {
		// If no document is found, then it's available.
		if err.Error() == "mongo: no documents in result" {
			return true, nil
		}
		return false, err
	}
	// Document found – provider details are taken.
	return false, nil
}
