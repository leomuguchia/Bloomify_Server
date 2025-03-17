package user

import (
	"bloomify/models"
	"fmt"
)

func (s *DefaultUserService) GetUserDevices(userID string) ([]models.Device, error) {
	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user.Devices, nil
}

func (s *DefaultUserService) SignOutOtherDevices(userID, currentDeviceID string) error {
	user, err := s.Repo.GetByIDWithProjection(userID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	filteredDevices := []models.Device{}
	for _, device := range user.Devices {
		if device.DeviceID == currentDeviceID {
			filteredDevices = append(filteredDevices, device)
		}
	}
	user.Devices = filteredDevices

	if err := s.Repo.Update(user); err != nil {
		return fmt.Errorf("failed to update user devices: %w", err)
	}

	return nil
}
