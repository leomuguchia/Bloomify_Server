package admin

import (
	"bloomify/models"
)

type AdminService interface {
	GetLegalSections() []models.LegalSection
	GetLegalSectionsFor(role string) []models.LegalSection
}

// DefaultUserService is the production implementation.
type DefaultAdminService struct{}
