package userRepo

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

// MongoUserRepo implements UserRepository using MongoDB.
type MongoUserRepo struct {
	coll *mongo.Collection
}

// NewMongoUserRepo creates a new instance of UserRepository using MongoDB.
func NewMongoUserRepo() UserRepository {
	coll := database.MongoClient.Database("bloomify").Collection("users")
	repo := &MongoUserRepo{coll: coll}

	if err := repo.ensureIndexes(); err != nil {
		fmt.Printf("failed to create indexes: %v\n", err)
	}
	return repo
}

// newContext creates a context with the given timeout.
func newContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ensureIndexes creates indexes for fields frequently used in queries.
func (r *MongoUserRepo) ensureIndexes() error {
	ctx, cancel := newContext(10 * time.Second)
	defer cancel()

	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
	}

	_, err := r.coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}

// --- Projection-based Helper Methods ---

// GetByIDWithProjection retrieves a user by its unique ID using a projection.
// Pass nil for projection to retrieve the full document.
func (r *MongoUserRepo) GetByIDWithProjection(id string, projection bson.M) (*models.User, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne()
	if projection != nil {
		opts.SetProjection(projection)
	}

	var user models.User
	if err := r.coll.FindOne(ctx, bson.M{"id": id}, opts).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to fetch user with id %s: %w", id, err)
	}
	return &user, nil
}

// GetByEmailWithProjection retrieves a user by its email using a projection.
// Pass nil for projection to retrieve the full document.
func (r *MongoUserRepo) GetByEmailWithProjection(email string, projection bson.M) (*models.User, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne()
	if projection != nil {
		opts.SetProjection(projection)
	}

	var user models.User
	if err := r.coll.FindOne(ctx, bson.M{"email": email}, opts).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch user with email %s: %w", email, err)
	}
	return &user, nil
}

// GetAllWithProjection retrieves all users with an optional projection.
func (r *MongoUserRepo) GetAllWithProjection(projection bson.M) ([]models.User, error) {
	ctx, cancel := newContext(10 * time.Second)
	defer cancel()

	opts := options.Find()
	if projection != nil {
		opts.SetProjection(projection)
	}

	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	for cursor.Next(ctx) {
		var u models.User
		if err := cursor.Decode(&u); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// --- Exported Methods that Satisfy the UserRepository Interface ---

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

// GetByID retrieves a user by its unique ID (full document).
func (r *MongoUserRepo) GetByID(id string) (*models.User, error) {
	return r.GetByIDWithProjection(id, nil)
}

// GetAll retrieves all users (full documents).
func (r *MongoUserRepo) GetAll() ([]models.User, error) {
	return r.GetAllWithProjection(nil)
}

// GetByEmail retrieves a user by its email address (full document).
func (r *MongoUserRepo) GetByEmail(email string) (*models.User, error) {
	return r.GetByEmailWithProjection(email, nil)
}
