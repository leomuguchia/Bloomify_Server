// File: database/repository/user/userMongoCrud.go
package userRepo

import (
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Create inserts a new user document.
func (r *MongoUserRepo) Create(user *models.User) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.coll.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Update modifies an existing user document.
func (r *MongoUserRepo) Update(user *models.User) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	user.UpdatedAt = time.Now()
	filter := bson.M{"id": user.ID}
	update := bson.M{"$set": user}

	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user with id %s: %w", user.ID, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", user.ID)
	}
	return nil
}

// Delete removes a user document by its ID.
func (r *MongoUserRepo) Delete(id string) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	result, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user with id %s: %w", id, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	return nil
}

func (r *MongoUserRepo) UpdateSetDocument(id string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Wrap in $set to comply with MongoDB update syntax
	update := bson.M{"$set": updateDoc}

	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	return nil
}

func (r *MongoUserRepo) UpdateAddToSetDocument(id string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Wrap in $addToSet to ensure uniqueness
	update := bson.M{"$addToSet": updateDoc}

	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	return nil
}
func (r *MongoUserRepo) PullFromArray(id, field string, value interface{}) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	var pullCondition interface{}

	// If value is a slice, use $in with that slice; otherwise pull the single value.
	switch v := value.(type) {
	case []interface{}:
		pullCondition = bson.M{"$in": v}
	default:
		pullCondition = v
	}

	update := bson.M{
		"$pull": bson.M{
			field: pullCondition,
		},
	}
	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to pull from %s for user %s: %w", field, id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	return nil
}

func (r *MongoUserRepo) MarkNotificationsAsRead(id string, notificationIDs []string) error {
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
	// 3. Update the user's updatedAt field
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
		return fmt.Errorf("failed to mark notifications as read for user %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("no notifications were updated (possibly already read or IDs not found)")
	}

	return nil
}
