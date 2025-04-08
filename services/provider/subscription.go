package provider

import (
	"fmt"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
)

// isAdvancedVerificationComplete checks if the provider's advanced verification data is complete.
func isAdvancedVerificationComplete(adv models.AdvancedVerification) bool {
	return len(adv.InsuranceDocs) > 0 && adv.TaxPIN != ""
}

// EnableSubscription verifies that the provider meets all the basic criteria and enables subscriptions.
func (ps *DefaultProviderService) EnableSubscription(providerID string) error {
	prov, err := ps.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve provider: %w", err)
	}

	if !isAdvancedVerificationComplete(prov.AdvancedVerification) {
		return fmt.Errorf("provider has not completed advanced verification")
	}

	// Provider must be active for at least 3 months.
	if time.Since(prov.CreatedAt) < (90 * 24 * time.Hour) {
		return fmt.Errorf("provider must be active for at least 3 months")
	}

	// Check performance criteria.
	if prov.Profile.Rating < 4.5 {
		return fmt.Errorf("provider rating too low")
	}
	if prov.CompletedBookings < 10 {
		return fmt.Errorf("provider must have completed at least 10 bookings")
	}

	// Enable subscriptions.
	updateFields := map[string]interface{}{
		"subscriptionEnabled": true,
	}
	if err := ps.Repo.UpdateWithDocument(providerID, updateFields); err != nil {
		return fmt.Errorf("failed to update provider subscription status: %w", err)
	}
	return nil
}

// UpdateSubscriptionSettings updates the provider's subscription configuration.
func (ps *DefaultProviderService) UpdateSubscriptionSettings(providerID string, settings models.SubscriptionModel) error {
	updateFields := map[string]interface{}{
		"subscriptionModel": settings,
	}
	if err := ps.Repo.UpdateWithDocument(providerID, updateFields); err != nil {
		return fmt.Errorf("failed to update subscription settings: %w", err)
	}
	return nil
}

func (ps *DefaultProviderService) GetSubscriptionHistory(providerID string) ([]models.SubscriptionBooking, error) {
	// Fetch only the subscriptionBooking field using a projection.
	prov, err := ps.Repo.GetByIDWithProjection(providerID, bson.M{"subscriptionBooking": 1})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider with subscription history: %w", err)
	}
	return prov.SubscriptionBooking, nil
}
