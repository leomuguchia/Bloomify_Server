package user

import (
	"bloomify/models"
	"fmt"
)

// GetAllUsers retrieves all users for admin access, excluding sensitive fields.
func (s *DefaultUserService) GetAllUsers() ([]models.User, error) {
	users, err := s.Repo.GetAllSafe()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	return users, nil
}
