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

// ensureIndexes creates indexes for fields that are frequently used in queries.
func (r *MongoProviderRepo) ensureIndexes() error {
	ctx, cancel := newContext(10 * time.Second)
	defer cancel()

	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "service_type", Value: 1}}},
		{Keys: bson.D{{Key: "location", Value: 1}}},
		// Create a 2dsphere index on the location_geo field for geospatial queries.
		{Keys: bson.D{{Key: "location_geo", Value: "2dsphere"}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}

// GetByTokenHash retrieves a provider by its token_hash using a projection.
func (r *MongoProviderRepo) GetByTokenHash(tokenHash string) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne().SetProjection(bson.M{"token_hash": 1, "id": 1})
	var result struct {
		ID        string `bson:"id"`
		TokenHash string `bson:"token_hash"`
	}
	if err := r.coll.FindOne(ctx, bson.M{"token_hash": tokenHash}, opts).Decode(&result); err != nil {
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
			"password_hash": 0,
			"token_hash":    0,
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
			"password_hash": 0,
			"token_hash":    0,
		}
	} else {
		proj = projection
	}

	opts := options.FindOne().SetProjection(proj)
	var provider models.Provider
	if err := r.coll.FindOne(ctx, bson.M{"email": email}, opts).Decode(&provider); err != nil {
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
			"password_hash": 0,
			"token_hash":    0,
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

	filter := bson.M{"service_type": bson.M{"$regex": service, "$options": "i"}}
	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"password_hash": 0,
			"token_hash":    0,
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
