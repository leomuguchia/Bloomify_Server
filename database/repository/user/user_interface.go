package userRepo

import (
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// UserRepository defines methods for user data access.
type UserRepository interface {
	// GetAll retrieves all users.
	GetAllSafe() ([]models.User, error)
	// Create inserts a new user record.
	Create(user *models.User) error
	// Update modifies an existing user record.
	Update(user *models.User) error
	// Delete removes a user record by its ID.
	Delete(id string) error
	// GetByIDWithProjection retrieves a user by its unique ID with a projection.
	GetByIDWithProjection(id string, projection bson.M) (*models.User, error)
	// GetByEmailWithProjection retrieves a user by its email with a projection.
	GetByEmailWithProjection(email string, projection bson.M) (*models.User, error)
	// GetAllWithProjection retrieves all users with an optional projection.
	GetAllWithProjection(projection bson.M) ([]models.User, error)
}
