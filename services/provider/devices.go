package provider

import (
	"fmt"

	"bloomify/models"
)

// GetProviderDevices retrieves the list of devices associated with a provider.
func (s *DefaultProviderService) GetProviderDevices(providerID string) ([]models.Device, error) {
	provider, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return provider.Devices, nil
}

// SignOutOtherDevices retains only the device matching the currentDeviceID for the provider.
func (s *DefaultProviderService) SignOutOtherDevices(providerID, currentDeviceID string) error {
	provider, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve provider: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("provider not found")
	}

	// Retain only the device that matches the currentDeviceID.
	filteredDevices := []models.Device{}
	for _, device := range provider.Devices {
		if device.DeviceID == currentDeviceID {
			filteredDevices = append(filteredDevices, device)
		}
	}
	provider.Devices = filteredDevices

	// Update the provider document in the repository.
	if err := s.Repo.Update(provider); err != nil {
		return fmt.Errorf("failed to update provider devices: %w", err)
	}
	return nil
}
