// File: database/repository/user/userMongoCrud.go
package userRepo

import (
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
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

// UpdateWithDocument updates a user using a custom update document.
func (r *MongoUserRepo) UpdateWithDocument(id string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	result, err := r.coll.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update user with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}
	return nil
}
