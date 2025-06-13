package providerRepo

import (
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// Delete removes a provider document by its ID.
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

func (r *MongoProviderRepo) UpdateSetDocument(id string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Wrap in $set to comply with MongoDB update syntax
	update := bson.M{"$set": updateDoc}

	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update provider with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("provider with id %s not found", id)
	}
	return nil
}

func (r *MongoProviderRepo) UpdatePushDocument(id string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Wrap in $set to comply with MongoDB update syntax
	update := bson.M{"$push": updateDoc}

	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update provider with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("provider with id %s not found", id)
	}
	return nil
}

func (r *MongoProviderRepo) MarkNotificationsAsRead(id string, notificationIDs []string) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Create array filters for the specific notifications to update
	arrayFilters := options.ArrayFilters{
		Filters: []any{
			bson.M{
				"elem.id": bson.M{"$in": notificationIDs},
			},
		},
	}

	// Update operation to:
	// 1. Set read=true for matching notifications
	// 2. Update their updatedAt timestamp
	// 3. Update the provider's updatedAt field
	update := bson.M{
		"$set": bson.M{
			"notifications.$[elem].read":      true,
			"notifications.$[elem].updatedAt": time.Now(),
			"updatedAt":                       time.Now(),
		},
	}

	opts := options.Update().SetArrayFilters(arrayFilters)
	filter := bson.M{"id": id}

	result, err := r.coll.UpdateOne(
		ctx,
		filter,
		update,
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to mark notifications as read for provider %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("provider with id %s not found", id)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("no notifications were updated (possibly already read or IDs not found)")
	}

	return nil
}
