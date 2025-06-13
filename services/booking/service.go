package booking

import (
	"bloomify/models"
	"bloomify/utils"
	"fmt"
	"math"
	"strings"
)

type PriceRange struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Suggested float64 `json:"suggested,omitempty"`
	Currency  string  `json:"currency"`
}

type ServiceDetails struct {
	Metadata      models.ServiceMetadata `json:"metadata"`
	PriceRange    *PriceRange            `json:"priceRange,omitempty"`
	CustomOptions []models.CustomOption  `json:"customOptions,omitempty"`
	Availability  []string               `json:"availability,omitempty"`
}

var globalRegions = []string{
	"East Asia & Pacific",
	"Europe & Central Asia",
	"Latin America & Caribbean",
	"Middle East & North Africa",
	"Sub-Saharan Africa",
	"South Asia",
	"North America",
}

// price range is in default currrency USD
var servicesMap = map[string]ServiceDetails{
	"Cleaning": {
		Metadata: models.ServiceMetadata{
			ID:           "Cleaning",
			Icon:         "🧹",
			UnitType:     "hours",
			ProviderTerm: "Cleaners",
			Modes:        []string{models.ModeInHome},
			Category:     "Domestic Services",
		},
		PriceRange: &PriceRange{Min: 20, Max: 40},
		CustomOptions: []models.CustomOption{
			{Option: "Deep Cleaning", Multiplier: 2.0},
			{Option: "Move-In/Move-Out", Multiplier: 1.8},
			{Option: "Pet Hair Removal", Multiplier: 1.5},
			{Option: "Eco-Friendly Products", Multiplier: 1.2},
		},
		Availability: globalRegions,
	},
	"Laundry": {
		Metadata: models.ServiceMetadata{
			ID:           "Laundry",
			Icon:         "🧺",
			UnitType:     "kgs",
			ProviderTerm: "Laundry Providers",
			Modes:        []string{models.ModePickupDelivery, models.ModeInHome},
			Category:     "Domestic Services",
		},
		PriceRange: &PriceRange{Min: 5, Max: 15},
		CustomOptions: []models.CustomOption{
			{Option: "Hypoallergenic Detergent", Multiplier: 1.3},
			{Option: "Delicates/Hand-Wash", Multiplier: 1.7},
			{Option: "Express Service", Multiplier: 2.0},
			{Option: "Folding Preferences", Multiplier: 1.2},
		},
		Availability: globalRegions,
	},
	"MealPrep": {
		Metadata: models.ServiceMetadata{
			ID:           "MealPrep",
			Icon:         "🍽️",
			UnitType:     "sessions",
			ProviderTerm: "Personal Chefs",
			Modes:        []string{models.ModeInHome, models.ModePickupDelivery},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 15, Max: 50},
		CustomOptions: []models.CustomOption{
			{Option: "Vegan/Vegetarian", Multiplier: 1.3},
			{Option: "Gluten-Free", Multiplier: 1.2},
			{Option: "Family Size Portions", Multiplier: 1.5},
			{Option: "In-Home Cooking", Multiplier: 2.0},
		},
		Availability: globalRegions,
	},
	"Handyman": {
		Metadata: models.ServiceMetadata{
			ID:           "Handyman",
			Icon:         "🔧",
			UnitType:     "hours",
			ProviderTerm: "Handymen",
			Modes:        []string{models.ModeInHome},
			Category:     "Domestic Services",
		},
		PriceRange: &PriceRange{Min: 30, Max: 60},
		CustomOptions: []models.CustomOption{
			{Option: "TV Mounting", Multiplier: 1.5},
			{Option: "Furniture Assembly", Multiplier: 1.3},
			{Option: "Wall Repairs", Multiplier: 1.4},
			{Option: "Emergency Repair", Multiplier: 2.0},
		},
		Availability: globalRegions,
	},
	"LawnCare": {
		Metadata: models.ServiceMetadata{
			ID:           "LawnCare",
			Icon:         "🌿",
			UnitType:     "hours",
			ProviderTerm: "Gardeners",
			Modes:        []string{models.ModeInHome},
			Category:     "Domestic Services",
		},
		PriceRange: &PriceRange{Min: 20, Max: 50},
		CustomOptions: []models.CustomOption{
			{Option: "Flower Bed Planting", Multiplier: 1.4},
			{Option: "Tree Trimming", Multiplier: 1.6},
			{Option: "Indoor Plant Maintenance", Multiplier: 1.3},
			{Option: "Biweekly Mowing", Multiplier: 1.2},
		},
		Availability: globalRegions,
	},
	"PetCare": {
		Metadata: models.ServiceMetadata{
			ID:           "PetCare",
			Icon:         "🐾",
			UnitType:     "hours",
			ProviderTerm: "Pet Sitters",
			Modes:        []string{models.ModeInHome},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 15, Max: 40},
		CustomOptions: []models.CustomOption{
			{Option: "Dog Walking", Multiplier: 1.0},
			{Option: "Pet Grooming", Multiplier: 1.8},
			{Option: "Overnight Stay", Multiplier: 2.5},
			{Option: "Multiple Pets", Multiplier: 1.3},
		},
		Availability: globalRegions,
	},
	"Childcare": {
		Metadata: models.ServiceMetadata{
			ID:           "Childcare",
			Icon:         "🧒",
			UnitType:     "children",
			ProviderTerm: "Nannies",
			Modes:        []string{models.ModeInHome, models.ModeInStore},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 12, Max: 30},
		CustomOptions: []models.CustomOption{
			{Option: "Homework Help", Multiplier: 1.4},
			{Option: "Special Needs Care", Multiplier: 2.0},
			{Option: "Late Night Hours", Multiplier: 1.8},
			{Option: "Meal Preparation", Multiplier: 1.3},
		},
		Availability: globalRegions,
	},
	"Tutoring": {
		Metadata: models.ServiceMetadata{
			ID:           "Tutoring",
			Icon:         "📚",
			UnitType:     "sessions",
			ProviderTerm: "Tutors",
			Modes:        []string{models.ModeInHome, models.ModeVirtual},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 20, Max: 60},
		CustomOptions: []models.CustomOption{
			{Option: "STEM Subjects", Multiplier: 1.5},
			{Option: "Test Prep", Multiplier: 1.6},
			{Option: "Group Session (2+ Students)", Multiplier: 1.2},
			{Option: "Special Education", Multiplier: 2.0},
		},
		Availability: globalRegions,
	},
	"TechSupport": {
		Metadata: models.ServiceMetadata{
			ID:           "TechSupport",
			Icon:         "🖥️",
			UnitType:     "sessions",
			ProviderTerm: "IT Technicians",
			Modes:        []string{models.ModeInHome, models.ModeInStore, models.ModeVirtual},
			Category:     "Professional/Office Services",
		},
		PriceRange: &PriceRange{Min: 30, Max: 70},
		CustomOptions: []models.CustomOption{
			{Option: "Wi-Fi Setup", Multiplier: 1.2},
			{Option: "Device Troubleshooting", Multiplier: 1.4},
			{Option: "Printer/Scanner Setup", Multiplier: 1.3},
			{Option: "Multiple Devices", Multiplier: 1.5},
		},
		Availability: globalRegions,
	},
	"ErrandRunner": {
		Metadata: models.ServiceMetadata{
			ID:           "ErrandRunner",
			Icon:         "🛍️",
			UnitType:     "hours",
			ProviderTerm: "Concierges",
			Modes:        []string{models.ModeInHome, models.ModePickupDelivery},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 15, Max: 35},
		CustomOptions: []models.CustomOption{
			{Option: "Return Items", Multiplier: 1.1},
			{Option: "Gift Shopping", Multiplier: 1.3},
			{Option: "Multiple Stops", Multiplier: 1.4},
			{Option: "Same-Day Completion", Multiplier: 1.5},
		},
		Availability: globalRegions,
	},
	"Chauffeuring": {
		Metadata: models.ServiceMetadata{
			ID:           "Chauffeuring",
			Icon:         "🚗",
			UnitType:     "hours",
			ProviderTerm: "Chaeuffeurs",
			Modes:        []string{models.ModeInHome},
			Category:     "Lifestyle & Personal Services",
		},
		PriceRange: &PriceRange{Min: 30, Max: 60},
		CustomOptions: []models.CustomOption{
			{Option: "Use My Vehicle", Multiplier: 1.0},
			{Option: "Event-Ready (Formal Attire)", Multiplier: 1.5},
			{Option: "Luxury Vehicle Handling Experience", Multiplier: 2.0},
			{Option: "Wait & Return Trip", Multiplier: 1.8},
			{Option: "Early/Late Hours (Before 6AM / After 10PM)", Multiplier: 1.4},
		},
		Availability: globalRegions,
	},
	"PersonalCare": {
		Metadata: models.ServiceMetadata{
			ID:           "PersonalCare",
			Icon:         "💇‍♂️",
			UnitType:     "sessions",
			ProviderTerm: "Personal Care Specialists",
			Modes:        []string{models.ModeInHome, models.ModeInStore},
			Category:     "Personal Care",
		},
		PriceRange: &PriceRange{Min: 25, Max: 70},
		CustomOptions: []models.CustomOption{
			{Option: "Haircut + Beard Trim", Multiplier: 1.3},
			{Option: "Hot Towel Shave or Facial", Multiplier: 1.5},
			{Option: "Manicure & Pedicure", Multiplier: 1.6},
			{Option: "In-Home Appointment", Multiplier: 1.8},
			{Option: "Special Occasion Styling (Event/Wedding)", Multiplier: 2.0},
		},
		Availability: globalRegions,
	},
}

// GetServicesMap returns the static map of all service details.
func GetServicesMap() map[string]ServiceDetails {
	return servicesMap
}

// GetAvailableServices returns all services metadata available for the specified region.
// If region == "", it returns all services.
func (svc *DefaultBookingSessionService) GetAvailableServices(region string) ([]models.ServiceMetadata, error) {
	services := make([]models.ServiceMetadata, 0, len(servicesMap))

	for _, details := range servicesMap {
		// If a region filter is provided, check availability
		if region != "" && region != "global" {
			found := false
			regionLower := strings.ToLower(region)
			for _, avail := range details.Availability {
				if strings.Contains(strings.ToLower(avail), regionLower) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Append service metadata (no filtering or region matched)
		svcMeta := details.Metadata
		services = append(services, models.ServiceMetadata{
			ID:           svcMeta.ID,
			Icon:         svcMeta.Icon,
			UnitType:     svcMeta.UnitType,
			ProviderTerm: svcMeta.ProviderTerm,
			Modes:        svcMeta.Modes,
			Category:     svcMeta.Category,
		})
	}

	return services, nil
}

func (svc *DefaultBookingSessionService) GetServiceByID(serviceID string, countryCode string, currency string) (*ServiceDetails, error) {
	origDetails, exists := servicesMap[serviceID]
	if !exists {
		return nil, fmt.Errorf("service with ID %s not found", serviceID)
	}

	details := origDetails
	if origDetails.CustomOptions != nil {
		details.CustomOptions = make([]models.CustomOption, len(origDetails.CustomOptions))
		copy(details.CustomOptions, origDetails.CustomOptions)
	}
	if origDetails.PriceRange != nil {
		details.PriceRange = &PriceRange{
			Min:       origDetails.PriceRange.Min,
			Max:       origDetails.PriceRange.Max,
			Suggested: origDetails.PriceRange.Suggested,
			Currency:  origDetails.PriceRange.Currency,
		}
	}

	// Set default currency if blank
	if currency == "" {
		currency = "USD"
	}

	// Apply geo pricing and convert currency
	if details.PriceRange != nil {
		bias := 0.5
		geoBias, err := utils.GetGeoPricingBias(countryCode)
		if err != nil {
			return nil, fmt.Errorf("invalid country code %s: %w", countryCode, err)
		}
		biasedMin := details.PriceRange.Min * geoBias
		biasedMax := details.PriceRange.Max * geoBias
		biasedSuggested := biasedMin + bias*(biasedMax-biasedMin)

		convertedMin, err := utils.ConvertCurrency(biasedMin, "USD", currency)
		if err != nil {
			return nil, fmt.Errorf("currency conversion failed: %v", err)
		}
		convertedMax, err := utils.ConvertCurrency(biasedMax, "USD", currency)
		if err != nil {
			return nil, fmt.Errorf("currency conversion failed: %v", err)
		}
		convertedSuggested, err := utils.ConvertCurrency(biasedSuggested, "USD", currency)
		if err != nil {
			return nil, fmt.Errorf("currency conversion failed: %v", err)
		}

		// Round all prices to nearest whole number
		roundedMin := math.Round(convertedMin)
		roundedMax := math.Round(convertedMax)
		roundedSuggested := math.Round(convertedSuggested)

		details.PriceRange = &PriceRange{
			Min:       roundedMin,
			Max:       roundedMax,
			Suggested: roundedSuggested,
			Currency:  currency,
		}
	}

	// Preserve original multipliers
	customOptionSuggestions := make([]models.CustomOption, 0, len(details.CustomOptions))
	for _, opt := range details.CustomOptions {
		customOptionSuggestions = append(customOptionSuggestions, models.CustomOption{
			Option:     opt.Option,
			Multiplier: opt.Multiplier,
		})
	}
	details.CustomOptions = customOptionSuggestions

	return &details, nil
}
