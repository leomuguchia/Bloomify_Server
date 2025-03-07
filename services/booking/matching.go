package booking

import (
	"context"
	"fmt"
	"math"
	"sort"

	"bloomify/database/repository"
	"bloomify/models"
)

// RankedProvider represents a provider along with its computed ranking details.
type RankedProvider struct {
	Provider   models.Provider // Underlying provider details.
	RankPoints float64         // Composite score from matching.
	Preferred  bool            // True if this provider is the top match.
}

// MatchingService defines methods to match providers based on a service plan.
type MatchingService interface {
	// MatchProviders returns a ranked list of providers for a given service plan.
	MatchProviders(plan models.ServicePlan) ([]models.Provider, error)
}

// DefaultMatchingService is our production-ready implementation.
type DefaultMatchingService struct {
	// ProviderRepo accesses provider data from the database.
	ProviderRepo repository.ProviderRepository
}

// MatchProviders focuses solely on finding and ranking providers based on the service plan.
func (s *DefaultMatchingService) MatchProviders(plan models.ServicePlan) ([]models.Provider, error) {
	ctx := context.Background()
	// Call our matching logic to get ranked providers.
	rankedProviders, err := s.matchProviders(plan, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to match providers: %w", err)
	}
	return extractProviders(rankedProviders), nil
}

// matchProviders contains the core matching logic to rank providers.
func (s *DefaultMatchingService) matchProviders(plan models.ServicePlan, ctx context.Context) ([]RankedProvider, error) {
	// Build advanced search criteria.
	criteria := repository.ProviderSearchCriteria{
		ServiceType:   plan.Service,
		Location:      plan.Location,
		Latitude:      plan.Latitude,
		Longitude:     plan.Longitude,
		MaxDistanceKm: 5, // Maximum effective distance.
	}
	// Perform advanced search to get candidate providers.
	providers, err := s.ProviderRepo.AdvancedSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("advanced search failed: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found for service '%s'", plan.Service)
	}

	// Define scoring constants.
	const (
		MaxLocationPoints = 45.0 // Proximity weight.
		VerifiedBonus     = 20.0 // Fixed bonus if verified.
		MaxCompletedPts   = 20.0 // Maximum points for completed bookings.
		MaxRatingPts      = 15.0 // Maximum points for rating.
	)

	// Helper: compute location score (linear: 0 km -> 45 points, 50 km -> 0 points).
	computeLocationScore := func(distanceKm float64) float64 {
		if distanceKm >= 50 {
			return 0
		}
		return MaxLocationPoints * (1 - distanceKm/50)
	}

	// Helper: compute completed bookings score.
	// We'll cap the benefit; assume 100 or more bookings yields max points.
	computeCompletedScore := func(completed int) float64 {
		if completed >= 100 {
			return MaxCompletedPts
		}
		return (float64(completed) / 100) * MaxCompletedPts
	}

	// Helper: compute rating score.
	computeRatingScore := func(rating float64) float64 {
		if rating > 5 {
			rating = 5
		}
		return (rating / 5) * MaxRatingPts
	}

	type scoreData struct {
		Provider       models.Provider
		TotalScore     float64
		LocationScore  float64
		VerifiedScore  float64
		CompletedScore float64
		RatingScore    float64
	}
	var scores []scoreData
	for _, p := range providers {
		// Compute distance using haversine.
		distanceKm := haversine(plan.Latitude, plan.Longitude, p.Latitude, p.Longitude)
		locScore := computeLocationScore(distanceKm)
		// Verified bonus.
		var verifiedScore float64
		if p.AdvancedVerified {
			verifiedScore = VerifiedBonus
		}
		// Completed bookings score.
		compScore := computeCompletedScore(p.CompletedBookings)
		// Rating score.
		ratingScore := computeRatingScore(p.Rating)
		totalScore := locScore + verifiedScore + compScore + ratingScore
		scores = append(scores, scoreData{
			Provider:       p,
			TotalScore:     totalScore,
			LocationScore:  locScore,
			VerifiedScore:  verifiedScore,
			CompletedScore: compScore,
			RatingScore:    ratingScore,
		})
	}

	// Sort candidates by total score in descending order.
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	// Ensure the top candidate is marked as preferred.
	var ranked []RankedProvider
	for i, sd := range scores {
		ranked = append(ranked, RankedProvider{
			Provider:   sd.Provider,
			RankPoints: sd.TotalScore,
			Preferred:  i == 0, // The top candidate.
		})
	}

	return ranked, nil
}

// haversine calculates the great-circle distance (in km) between two geographic coordinates.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers.
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	lat1Rad := lat1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	a := (math.Sin(dLat/2) * math.Sin(dLat/2)) +
		(math.Cos(lat1Rad) * math.Cos(lat2Rad) * math.Sin(dLon/2) * math.Sin(dLon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// extractProviders converts a slice of RankedProvider to a slice of models.Provider.
func extractProviders(ranked []RankedProvider) []models.Provider {
	var providers []models.Provider
	for _, rp := range ranked {
		providers = append(providers, rp.Provider)
	}
	return providers
}
