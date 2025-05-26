package admin

import (
	"bloomify/models"
	"time"
)

// GetLegalSections returns all legal documents.
func (a *DefaultAdminService) GetLegalSections() []models.LegalSection {
	now := time.Now().UTC().Format(time.RFC3339)

	return []models.LegalSection{
		{
			ID:       "tos",
			Title:    "Terms of Service",
			Summary:  "These terms govern your use of the Bloomify platform.",
			Content:  generateTermsOfService(),
			Category: models.RoleUser,
			Version:  "v1.0",
			Updated:  now,
		},
		{
			ID:       "privacy",
			Title:    "Privacy Policy",
			Summary:  "How Bloomify collects and uses personal data.",
			Content:  generatePrivacyPolicy(),
			Category: models.RoleUser,
			Version:  "v1.0",
			Updated:  now,
		},
		{
			ID:       "conduct",
			Title:    "Community Guidelines & Code of Conduct",
			Summary:  "Rules all users and Bloomers must follow to ensure a safe experience.",
			Content:  generateCodeOfConduct(),
			Category: models.RoleBoth,
			Version:  "v1.0",
			Updated:  now,
		},
		{
			ID:       "payments",
			Title:    "Payment & Cancellation Policy",
			Summary:  "How payments, refunds, and cancellations work on Bloomify.",
			Content:  generatePaymentPolicy(),
			Category: models.RoleBoth,
			Version:  "v1.0",
			Updated:  now,
		},
	}
}

// GetLegalSectionsFor returns legal documents relevant to the specified role.
func (a *DefaultAdminService) GetLegalSectionsFor(role string) []models.LegalSection {
	all := a.GetLegalSections()
	var filtered []models.LegalSection

	for _, section := range all {
		if section.Category == models.RoleBoth || section.Category == role {
			filtered = append(filtered, section)
		}
	}
	return filtered
}

func generateTermsOfService() string {
	return `Welcome to Bloomify. By accessing or using our platform, you agree to be bound by these Terms of Service...

1. Eligibility: You must be 18+ to use Bloomify.
2. Platform Use: Bloomify connects users with independent service providers (“Bloomers”).
3. Liability: Bloomify is a facilitator; providers are independent.
4. Payments: Payments are processed via Stripe or agreed offline methods.
5. Cancellations: Each booking may have a different cancellation policy.
6. Disputes: Disputes must be reported within 48 hours after service.

Full details available on our website.`
}

func generatePrivacyPolicy() string {
	return `Bloomify values your privacy. We collect minimal personal data only as required to provide you with a seamless experience...

1. Data We Collect: Name, email, location, payment info.
2. How We Use It: Matching, billing, communication.
3. Third Parties: Stripe (payments), analytics.
4. Rights: You can request data deletion anytime.

See our full privacy terms online.`
}

func generateCodeOfConduct() string {
	return `All Bloomify users and Bloomers agree to:

- Be respectful and professional.
- Avoid discriminatory or harassing behavior.
- Respect time and privacy of others.
- Follow all applicable laws.

Violations may result in suspension or removal.`
}

func generatePaymentPolicy() string {
	return `1. Payments are securely processed via Stripe.
2. Bloomers may offer alternate payment (cash) if listed.
3. Users are charged upon booking confirmation.
4. Cancellations within 24 hours of service may incur a fee.
5. Refunds are issued for no-shows or service failures (on review).`
}
