package userRepo

import (
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// UserRepository defines methods for accessing and managing user data.
type UserRepository interface {
	// GetAllSafe retrieves all users, excluding sensitive information.
	GetAllSafe() ([]models.User, error)
	// Create inserts a new user record.
	Create(user *models.User) error
	// Update modifies an existing user record.
	Update(user *models.User) error
	// UpdateWithDocument updates a user record using an explicit update document.
	UpdateWithDocument(id string, updateDoc bson.M) error
	// Delete removes a user record by its ID.
	Delete(id string) error
	// GetByIDWithProjection retrieves a user by its unique ID using the specified projection.
	GetByIDWithProjection(id string, projection bson.M) (*models.User, error)
	// GetByEmailWithProjection retrieves a user by its email using the specified projection.
	GetByEmailWithProjection(email string, projection bson.M) (*models.User, error)
	// GetAllWithProjection retrieves all users using the specified projection.
	GetAllWithProjection(projection bson.M) ([]models.User, error)
}
