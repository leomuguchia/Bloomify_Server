package userRepo

import (
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// UserRepository defines methods for user data access.
type UserRepository interface {
	// GetByID retrieves a user by its unique ID.
	GetByID(id string) (*models.User, error)
	// GetAll retrieves all users.
	GetAll() ([]models.User, error)
	// GetByEmail retrieves a user by its email address.
	GetByEmail(email string) (*models.User, error)
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
