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
	// Use the "bloomify" database and the "providers" collection.
	coll := database.MongoClient.Database("bloomify").Collection("providers")
	return &MongoProviderRepo{coll: coll}
}

func (r *MongoProviderRepo) GetByID(id string) (*models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var provider models.Provider
	filter := bson.M{"id": id}
	if err := r.coll.FindOne(ctx, filter).Decode(&provider); err != nil {
		return nil, fmt.Errorf("failed to fetch provider with id %s: %w", id, err)
	}
	return &provider, nil
}

func (r *MongoProviderRepo) GetAll() ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := r.coll.Find(ctx, bson.M{})
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

func (r *MongoProviderRepo) GetByServiceType(service string) ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{
		"service_type": bson.M{"$regex": service, "$options": "i"},
	}
	cursor, err := r.coll.Find(ctx, filter)
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

func (r *MongoProviderRepo) Create(provider *models.Provider) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.coll.InsertOne(ctx, provider)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

func (r *MongoProviderRepo) Update(provider *models.Provider) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func (r *MongoProviderRepo) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// AdvancedSearch performs an advanced search based on the provided criteria.
// It returns at most 20 providers. Only providers with status "active" or "online"
// and non-zero latitude/longitude are returned.
func (r *MongoProviderRepo) AdvancedSearch(criteria ProviderSearchCriteria) ([]models.Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build the filter document based on criteria.
	filter := bson.M{}
	if criteria.ServiceType != "" {
		filter["service_type"] = bson.M{"$regex": criteria.ServiceType, "$options": "i"}
	}
	if criteria.Location != "" {
		filter["location"] = bson.M{"$regex": criteria.Location, "$options": "i"}
	}
	if criteria.MinRating > 0 {
		filter["rating"] = bson.M{"$gte": criteria.MinRating}
	}
	if criteria.MinCompletedBookings > 0 {
		filter["completed_bookings"] = bson.M{"$gte": criteria.MinCompletedBookings}
	}
	// For geospatial search, assume documents have a GeoJSON field "location_geo"
	if criteria.MaxDistanceKm > 0 {
		maxDistanceMeters := criteria.MaxDistanceKm * 1000
		filter["location_geo"] = bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{criteria.Longitude, criteria.Latitude},
				},
				"$maxDistance": maxDistanceMeters,
			},
		}
	}
	// Ensure provider is active/online and has a valid location.
	filter["status"] = bson.M{"$in": []string{"active", "online"}}
	filter["latitude"] = bson.M{"$ne": 0}
	filter["longitude"] = bson.M{"$ne": 0}

	// Set options: sort by rating (descending) and limit to 20 results.
	opts := options.Find().
		SetSort(bson.D{{Key: "rating", Value: -1}}).
		SetLimit(20)

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("advanced search query failed: %w", err)
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
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found matching criteria")
	}

	return providers, nil
}
