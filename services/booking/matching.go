package booking

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"bloomify/database/repository"
	"bloomify/models"
)

// Algorithmic matching service for providers based on service plans and proximity.
// This service matches providers to service plans based on various criteria
// such as service type, mode, and geographical proximity.
// It also provides a method to find nearby providers based on a given location.
// RankedProvider holds provider data along with computed score and proximity.
type RankedProvider struct {
	Provider       models.Provider
	RankPoints     float64
	Preferred      bool
	Proximity      float64
	ScoreBreakdown map[string]float64
}

// MatchingService defines the interface for matching providers.
type MatchingService interface {
	MatchProviders(plan models.ServicePlan) ([]models.ProviderDTO, error)
	MatchNearbyProviders(location models.GeoPoint) ([]models.ProviderDTO, error) // NEW
}

// DefaultMatchingService implements MatchingService.
type DefaultMatchingService struct {
	ProviderRepo repository.ProviderRepository
}

// MatchProviders receives a service plan, performs matching and returns provider DTOs.
// When no providers match, it returns an empty list rather than an error.
func (s *DefaultMatchingService) MatchProviders(plan models.ServicePlan) ([]models.ProviderDTO, error) {
	log.Printf("Received ServicePlan: %+v", plan)
	criteria := repository.ProviderSearchCriteria{
		ServiceType:   plan.ServiceType,
		Modes:         []string{plan.Mode},
		MaxDistanceKm: 5,
		LocationGeo:   plan.LocationGeo,
	}
	rankedProviders, err := s.matchProviders(criteria, context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to match providers: %w", err)
	}
	// If no providers are matched, return an empty list.
	if len(rankedProviders) == 0 {
		log.Printf("No providers matched for service '%s'", plan.ServiceType)
		return []models.ProviderDTO{}, nil
	}
	return extractProvidersDTO(rankedProviders), nil
}

func (s *DefaultMatchingService) matchProviders(
	criteria repository.ProviderSearchCriteria,
	ctx context.Context,
) ([]RankedProvider, error) {
	providers, err := s.ProviderRepo.AdvancedSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("advanced search failed: %w", err)
	}
	if len(providers) == 0 {
		return nil, nil
	}
	if len(criteria.LocationGeo.Coordinates) < 2 {
		return nil, fmt.Errorf("invalid center coords")
	}
	centerLon := criteria.LocationGeo.Coordinates[0]
	centerLat := criteria.LocationGeo.Coordinates[1]

	// Score + rank top 20
	return scoreAndRankProviders(providers, centerLat, centerLon, 20), nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	lat1Rad := lat1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (s *DefaultMatchingService) MatchNearbyProviders(
	location models.GeoPoint,
) ([]models.ProviderDTO, error) {
	criteria := repository.ProviderSearchCriteria{
		LocationGeo:   location,
		MaxDistanceKm: 5,
		Modes:         []string{"in_store", "pickup_delivery"},
	}
	// Reuse matchProviders (which in turn does AdvancedSearch + scoring)
	ranked, err := s.matchProviders(criteria, context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby providers: %w", err)
	}

	// Map RankedProvider → ProviderDTO
	var dtos []models.ProviderDTO
	for _, rp := range ranked {
		p := rp.Provider
		dtos = append(dtos, models.ProviderDTO{
			ID:               p.ID,
			Profile:          p.Profile,
			ServiceCatalogue: p.ServiceCatalogue,
			LocationGeo:      p.Profile.LocationGeo,
			Preferred:        rp.Preferred,
			Proximity:        rp.Proximity,
		})
	}
	return dtos, nil
}

func extractProvidersDTO(ranked []RankedProvider) []models.ProviderDTO {
	var dtos []models.ProviderDTO
	for _, rp := range ranked {
		dto := models.ProviderDTO{
			ID:               rp.Provider.ID,
			Profile:          rp.Provider.Profile,
			ServiceCatalogue: rp.Provider.ServiceCatalogue,
			LocationGeo:      rp.Provider.Profile.LocationGeo,
			Preferred:        rp.Preferred,
			Proximity:        rp.Proximity,
		}
		dtos = append(dtos, dto)
	}
	return dtos
}

// scoreAndRankProviders applies your weighted formula and returns top N
func scoreAndRankProviders(
	providers []models.Provider,
	centerLat, centerLon float64,
	maxResults int,
) []RankedProvider {
	const (
		MaxLocationPoints = 30.0
		VerifiedBonus     = 15.0
		MaxCompletedPts   = 25.0
		MaxRatingPts      = 20.0
		MaxSlotPts        = 10.0
		MaxDistanceKm     = 5.0
	)

	computeLocationScore := func(distKm float64) float64 {
		if distKm >= MaxDistanceKm {
			return 0
		}
		return MaxLocationPoints * (1 - distKm/MaxDistanceKm)
	}
	computeCompletedScore := func(c int) float64 {
		return math.Log10(float64(c+1)) * MaxCompletedPts / math.Log10(101)
	}
	computeRatingScore := func(r float64) float64 {
		if r > 5 {
			r = 5
		}
		return (r / 5) * MaxRatingPts
	}
	computeSlotScore := func(count int) float64 {
		if count >= 20 {
			return MaxSlotPts
		}
		return float64(count) / 20.0 * MaxSlotPts
	}

	type scoreData struct {
		sd RankedProvider
	}
	ch := make(chan RankedProvider, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)
		go func(p models.Provider) {
			defer wg.Done()
			lon, lat := 0.0, 0.0
			if len(p.Profile.LocationGeo.Coordinates) >= 2 {
				lon = p.Profile.LocationGeo.Coordinates[0]
				lat = p.Profile.LocationGeo.Coordinates[1]
			}
			distKm := haversine(centerLat, centerLon, lat, lon)
			locS := computeLocationScore(distKm)
			verS := 0.0
			if p.Profile.AdvancedVerified {
				verS = VerifiedBonus
			}
			compS := computeCompletedScore(p.CompletedBookings)
			ratS := computeRatingScore(p.Profile.Rating)
			slotCount := len(p.TimeSlotRefs)
			slotS := computeSlotScore(slotCount)
			total := locS + verS + compS + ratS + slotS

			ch <- RankedProvider{
				Provider:   p,
				RankPoints: total,
				Proximity:  distKm * 1000,
				ScoreBreakdown: map[string]float64{
					"proximity": locS,
					"verified":  verS,
					"completed": compS,
					"rating":    ratS,
					"slots":     slotS,
				},
			}
		}(p)
	}

	wg.Wait()
	close(ch)

	var scored []RankedProvider
	for rp := range ch {
		scored = append(scored, rp)
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].RankPoints > scored[j].RankPoints
	})

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}
	for i := range scored {
		scored[i].Preferred = (i == 0)
	}
	return scored
}
