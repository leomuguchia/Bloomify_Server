package userRepo

import (
	"bloomify/database"
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepository defines methods for accessing and managing user data.
type UserRepository interface {
	GetAllSafe() ([]models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	UpdateSetDocument(id string, updateDoc bson.M) error
	UpdateAddToSetDocument(id string, updateDoc bson.M) error
	Delete(id string) error
	GetByIDWithProjection(id string, projection bson.M) (*models.User, error)
	GetByEmailWithProjection(email string, projection bson.M) (*models.User, error)
	GetAllWithProjection(projection bson.M) ([]models.User, error)
	IsUserAvailable(basicReq models.UserBasicRegistrationData) (bool, error)
	PullFromArray(id string, field string, value interface{}) error
	MarkNotificationsAsRead(id string, notificationIDs []string) error
}

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
