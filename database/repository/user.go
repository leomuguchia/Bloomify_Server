// database/repository/user.go
package repository

import (
	"fmt"

	"bloomify/database"
	"bloomify/models"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id uint) error
}

// GormUserRepo implements UserRepository using GORM.
type GormUserRepo struct{}

// GetByID retrieves a user by their ID.
func (repo *GormUserRepo) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := database.DB.First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with id %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to retrieve user with id %d: %w", id, err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email.
func (repo *GormUserRepo) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := database.DB.First(&user, "email = ?", email).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with email %s not found: %w", email, err)
		}
		return nil, fmt.Errorf("failed to retrieve user with email %s: %w", email, err)
	}
	return &user, nil
}

// Create inserts a new user record into the database.
func (repo *GormUserRepo) Create(user *models.User) error {
	if err := database.DB.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Update saves the updated user record.
func (repo *GormUserRepo) Update(user *models.User) error {
	if err := database.DB.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user with id %d: %w", user.ID, err)
	}
	return nil
}

// Delete removes the user record by ID.
func (repo *GormUserRepo) Delete(id uint) error {
	if err := database.DB.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete user with id %d: %w", id, err)
	}
	return nil
}
