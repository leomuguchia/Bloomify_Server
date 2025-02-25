package userRepo

import "bloomify/models"

// UserRepository defines methods for user data access.
type UserRepository interface {
	// GetByID retrieves a user by their unique ID.
	GetByID(id uint) (*models.User, error)
	// GetByEmail retrieves a user by their email.
	GetByEmail(email string) (*models.User, error)
	// Create inserts a new user record.
	Create(user *models.User) error
	// Update modifies an existing user record.
	Update(user *models.User) error
	// Delete removes a user record by its ID.
	Delete(id uint) error
	// AdvancedSearch searches users based on given criteria.
	AdvancedSearch(criteria UserSearchCriteria) ([]models.User, error)
}

// UserSearchCriteria holds parameters for an advanced user search.
type UserSearchCriteria struct {
	Name      string  // Partial or full name, case-insensitive.
	Email     string  // Partial or full email, case-insensitive.
	MinRating float64 // For example, if you track user ratings.
}
