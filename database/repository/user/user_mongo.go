package userRepo

import (
	"context"
	"fmt"
	"time"

	"bloomify/database"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoUserRepo implements UserRepository using MongoDB.
type MongoUserRepo struct {
	coll *mongo.Collection
}

// NewMongoUserRepository constructs a new UserRepository using MongoDB.
func NewMongoUserRepository() UserRepository {
	coll := database.MongoClient.Database("bloomify").Collection("users")
	return &MongoUserRepo{coll: coll}
}

func (r *MongoUserRepo) GetByID(id uint) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	filter := bson.M{"id": id}
	if err := r.coll.FindOne(ctx, filter).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to fetch user with id %d: %w", id, err)
	}
	return &user, nil
}

func (r *MongoUserRepo) GetByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	filter := bson.M{"email": bson.M{"$regex": email, "$options": "i"}}
	if err := r.coll.FindOne(ctx, filter).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to fetch user with email %s: %w", email, err)
	}
	return &user, nil
}

func (r *MongoUserRepo) Create(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.coll.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *MongoUserRepo) Update(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": user.ID}
	update := bson.M{"$set": user}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user with id %d: %w", user.ID, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %d not found", user.ID)
	}
	return nil
}

func (r *MongoUserRepo) Delete(id uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	result, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user with id %d: %w", id, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}
	return nil
}

func (r *MongoUserRepo) AdvancedSearch(criteria UserSearchCriteria) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}
	if criteria.Name != "" {
		filter["name"] = bson.M{"$regex": criteria.Name, "$options": "i"}
	}
	if criteria.Email != "" {
		filter["email"] = bson.M{"$regex": criteria.Email, "$options": "i"}
	}
	// For example, if you want to filter by minimum rating.
	if criteria.MinRating > 0 {
		filter["rating"] = bson.M{"$gte": criteria.MinRating}
	}

	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("advanced search query failed: %w", err)
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
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return users, nil
}
