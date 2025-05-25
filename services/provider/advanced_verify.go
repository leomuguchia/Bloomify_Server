// File: bloomify/service/provider/provider.go
package provider

import (
	"context"
	"fmt"
	"time"

	"bloomify/models"
)

// AdvanceVerifyRequest represents the extra fields required for advanced verification.
type AdvanceVerifyRequest struct {
	TaxPIN        string   `json:"tax_pin" binding:"required"`        // Business tax ID; required for advanced verification
	InsuranceDocs []string `json:"insurance_docs" binding:"required"` // Insurance and certification docs; at least one required
}

// AdvanceVerifyProvider verifies extra advanced details for a provider.
// It updates the provider record to mark it as advanced verified and returns the updated provider details.
func (s *DefaultProviderService) AdvanceVerifyProvider(c context.Context, providerID string, advReq AdvanceVerifyRequest, fullAccess bool) (*models.Provider, error) {
	// Retrieve the current provider record without any projection restrictions.
	provider, err := s.Repo.GetByIDWithProjection(providerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Check if provider is already advanced verified.
	if provider.Profile.AdvancedVerified {
		return nil, fmt.Errorf("provider is already advanced verified")
	}

	// Validate extra fields: require a non-empty TaxPIN and at least one insurance document.
	if advReq.TaxPIN == "" {
		return nil, fmt.Errorf("tax PIN is required for advanced verification")
	}
	if len(advReq.InsuranceDocs) == 0 {
		return nil, fmt.Errorf("at least one insurance document is required for advanced verification")
	}

	// Update provider record with advanced verification details.
	provider.AdvancedVerification.TaxPIN = advReq.TaxPIN
	provider.AdvancedVerification.InsuranceDocs = advReq.InsuranceDocs
	provider.Profile.AdvancedVerified = true
	provider.VerificationLevel = "advanced"
	provider.UpdatedAt = time.Now()

	// Persist the updated provider record.
	if err := s.Repo.Update(provider); err != nil {
		return nil, fmt.Errorf("failed to update provider for advanced verification: %w", err)
	}

	// Retrieve the full provider details using the context flag.
	updatedProvider, err := s.GetProviderByID(c, providerID, fullAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated provider: %w", err)
	}
	return updatedProvider, nil
}
