// File: database/repository/user/userMongoQueries.go
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

// GetByIDWithProjection retrieves a user by its ID with an optional projection.
func (r *MongoUserRepo) GetByIDWithProjection(id string, projection bson.M) (*models.User, error) {
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
	var user models.User
	if err := r.coll.FindOne(ctx, bson.M{"id": id}, opts).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to fetch user with id %s: %w", id, err)
	}

	// If Devices field is nil, initialize it to an empty slice.
	if user.Devices == nil {
		user.Devices = []models.Device{}
	}

	return &user, nil
}

// GetByEmailWithProjection retrieves a user by its email using a projection.
func (r *MongoUserRepo) GetByEmailWithProjection(email string, projection bson.M) (*models.User, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	opts := options.FindOne()
	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}
	opts.SetProjection(proj)

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
	var proj bson.M
	if projection == nil {
		proj = bson.M{
			"passwordHash": 0,
			"tokenHash":    0,
		}
	} else {
		proj = projection
	}
	opts.SetProjection(proj)

	// Provide an empty filter to match all documents.
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

// GetAllSafe retrieves all users while excluding sensitive fields.
func (r *MongoUserRepo) GetAllSafe() ([]models.User, error) {
	projection := bson.M{"passwordHash": 0, "tokenHash": 0}
	return r.GetAllWithProjection(projection)
}

// IsUserAvailable checks whether a user with the given username or email already exists.
func (r *MongoUserRepo) IsUserAvailable(basicReq models.UserBasicRegistrationData) (bool, error) {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	// Create a filter that checks for the given username or email.
	filter := bson.M{
		"$or": []bson.M{
			{"username": basicReq.Username},
			{"email": basicReq.Email},
		},
	}
	var user models.User
	err := r.coll.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		// If no document was found, it's available.
		if err.Error() == "mongo: no documents in result" {
			return true, nil
		}
		// Otherwise, return the error.
		return false, err
	}
	// Document found – username or email is taken.
	return false, nil
}
