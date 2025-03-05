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
		{Keys: bson.D{{Key: "rating", Value: -1}}},
		{Keys: bson.D{{Key: "completed_bookings", Value: -1}}},
		{Keys: bson.D{{Key: "location_geo", Value: "2dsphere"}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "latitude", Value: 1}}},
		{Keys: bson.D{{Key: "longitude", Value: 1}}},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}

// --- Projection-based Helper Methods ---

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
			return nil, nil // No provider found for this token
		}
		return nil, fmt.Errorf("failed to retrieve provider by token hash: %w", err)
	}

	// If needed, you can fetch the full provider record here using result.ID.
	return r.GetByID(result.ID)
}

// GetByIDWithProjection retrieves a provider by its unique ID using a projection.
// Pass nil for projection if you want the full document.
func (r *MongoProviderRepo) GetByIDWithProjection(id string, projection bson.M) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne()
	if projection != nil {
		opts.SetProjection(projection)
	}

	var provider models.Provider
	if err := r.coll.FindOne(ctx, bson.M{"id": id}, opts).Decode(&provider); err != nil {
		return nil, fmt.Errorf("failed to fetch provider with id %s: %w", id, err)
	}
	return &provider, nil
}

// GetByEmailWithProjection retrieves a provider by its email using a projection.
// Pass nil for projection if you want the full document.
func (r *MongoProviderRepo) GetByEmailWithProjection(email string, projection bson.M) (*models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne()
	if projection != nil {
		opts.SetProjection(projection)
	}

	var provider models.Provider
	if err := r.coll.FindOne(ctx, bson.M{"email": email}, opts).Decode(&provider); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch provider with email %s: %w", email, err)
	}
	return &provider, nil
}

// GetAllWithProjection retrieves all providers with an optional projection.
func (r *MongoProviderRepo) GetAllWithProjection(projection bson.M) ([]models.Provider, error) {
	ctx, cancel := newContext(10 * time.Second)
	defer cancel()

	opts := options.Find()
	if projection != nil {
		opts.SetProjection(projection)
	}

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
		providers = append(providers, p)
	}
	return providers, nil
}

// GetByServiceTypeWithProjection retrieves providers matching the given service type with a projection.
func (r *MongoProviderRepo) GetByServiceTypeWithProjection(service string, projection bson.M) ([]models.Provider, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	filter := bson.M{"service_type": bson.M{"$regex": service, "$options": "i"}}
	opts := options.Find()
	if projection != nil {
		opts.SetProjection(projection)
	}

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
		providers = append(providers, p)
	}
	return providers, nil
}

// --- Exported Methods that Satisfy the ProviderRepository Interface ---

// GetByID retrieves a provider by its unique ID (full document).
func (r *MongoProviderRepo) GetByID(id string) (*models.Provider, error) {
	return r.GetByIDWithProjection(id, nil)
}

// GetAll retrieves all providers (full documents).
func (r *MongoProviderRepo) GetAll() ([]models.Provider, error) {
	return r.GetAllWithProjection(nil)
}

// GetByServiceType returns providers that offer a specific service (full documents).
func (r *MongoProviderRepo) GetByServiceType(service string) ([]models.Provider, error) {
	return r.GetByServiceTypeWithProjection(service, nil)
}

// GetByEmail retrieves a provider by its email address (full document).
func (r *MongoProviderRepo) GetByEmail(email string) (*models.Provider, error) {
	return r.GetByEmailWithProjection(email, nil)
}

// Create inserts a new provider document.
func (r *MongoProviderRepo) Create(provider *models.Provider) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	_, err := r.coll.InsertOne(ctx, provider)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

// Update modifies an existing provider document.
func (r *MongoProviderRepo) Update(provider *models.Provider) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	filter := bson.M{"id": provider.ID}
	update := bson.M{"$set": provider}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update provider with id %s: %w", provider.ID, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("provider with id %s not found", provider.ID)
	}
	return nil
}

// Delete removes a provider document.
func (r *MongoProviderRepo) Delete(id string) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	result, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete provider with id %s: %w", id, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("provider with id %s not found", id)
	}
	return nil
}
