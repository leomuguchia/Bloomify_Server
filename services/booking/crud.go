package booking

import "bloomify/models"

// GetAvailableServices returns a list of 10 available services stored in memory.
func (svc *DefaultBookingSessionService) GetAvailableServices() ([]models.Service, error) {
	services := []models.Service{
		{ID: "1", Name: "Babysitting", Icon: "people"},
		{ID: "2", Name: "Chauffeuring", Icon: "car"},
		{ID: "3", Name: "Laundry", Icon: "water"},
		{ID: "4", Name: "Cleaning", Icon: "broom"},
		{ID: "5", Name: "Plumbing", Icon: "construct"},
		{ID: "6", Name: "Electrical", Icon: "flash"},
		{ID: "7", Name: "Delivery", Icon: "cart"},
		{ID: "8", Name: "Pet Sitting", Icon: "paw"},
		{ID: "9", Name: "Tutoring", Icon: "book"},
		{ID: "10", Name: "Fitness Training", Icon: "fitness"},
	}
	return services, nil
}
