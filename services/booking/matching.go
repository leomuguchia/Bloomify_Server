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

// RankedProvider holds provider data along with computed score and proximity.
type RankedProvider struct {
	Provider   models.Provider
	RankPoints float64
	Preferred  bool
	Proximity  float64
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
		Mode:          plan.Mode,
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

// matchProviders performs the actual provider search, scoring and ranking.
func (s *DefaultMatchingService) matchProviders(criteria repository.ProviderSearchCriteria, ctx context.Context) ([]RankedProvider, error) {
	providers, err := s.ProviderRepo.AdvancedSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("advanced search failed: %w", err)
	}
	// If no providers are found at all, return an empty slice.
	if len(providers) == 0 {
		return []RankedProvider{}, nil
	}

	if len(criteria.LocationGeo.Coordinates) < 2 {
		return nil, fmt.Errorf("invalid search center coordinates")
	}
	centerLon := criteria.LocationGeo.Coordinates[0]
	centerLat := criteria.LocationGeo.Coordinates[1]

	const (
		MaxLocationPoints = 45.0
		VerifiedBonus     = 20.0
		MaxCompletedPts   = 20.0
		MaxRatingPts      = 15.0
	)

	computeLocationScore := func(distanceKm float64) float64 {
		if distanceKm >= 5 {
			return 0
		}
		return MaxLocationPoints * (1 - distanceKm/5)
	}
	computeCompletedScore := func(completed int) float64 {
		return math.Log10(float64(completed+1)) * MaxCompletedPts / math.Log10(101)
	}
	computeRatingScore := func(rating float64) float64 {
		if rating > 5 {
			rating = 5
		}
		return (rating / 5) * MaxRatingPts
	}

	// scoreData holds temporary scoring details.
	type scoreData struct {
		Provider       models.Provider
		TotalScore     float64
		LocationScore  float64
		VerifiedScore  float64
		CompletedScore float64
		RatingScore    float64
		DistanceKm     float64 // distance in km
	}

	resultsCh := make(chan scoreData, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)
		go func(p models.Provider) {
			defer wg.Done()
			var provLat, provLon float64
			if len(p.Profile.LocationGeo.Coordinates) >= 2 {
				provLon = p.Profile.LocationGeo.Coordinates[0]
				provLat = p.Profile.LocationGeo.Coordinates[1]
			}
			// Compute distance in km.
			distanceKm := haversine(centerLat, centerLon, provLat, provLon)
			locScore := computeLocationScore(distanceKm)
			var verifiedScore float64
			if p.Profile.AdvancedVerified {
				verifiedScore = VerifiedBonus
			}
			compScore := computeCompletedScore(p.CompletedBookings)
			ratingScore := computeRatingScore(p.Profile.Rating)
			totalScore := locScore + verifiedScore + compScore + ratingScore

			resultsCh <- scoreData{
				Provider:       p,
				TotalScore:     totalScore,
				LocationScore:  locScore,
				VerifiedScore:  verifiedScore,
				CompletedScore: compScore,
				RatingScore:    ratingScore,
				DistanceKm:     distanceKm,
			}
		}(p)
	}

	wg.Wait()
	close(resultsCh)

	var scores []scoreData
	for s := range resultsCh {
		scores = append(scores, s)
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	var ranked []RankedProvider
	for i, sd := range scores {
		ranked = append(ranked, RankedProvider{
			Provider:   sd.Provider,
			RankPoints: sd.TotalScore,
			Preferred:  i == 0,
			// Convert km to metres.
			Proximity: sd.DistanceKm * 1000,
		})
	}
	if len(ranked) > 20 {
		ranked = ranked[:20]
	}

	return ranked, nil
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

func (s *DefaultMatchingService) MatchNearbyProviders(location models.GeoPoint) ([]models.ProviderDTO, error) {
	criteria := repository.ProviderSearchCriteria{
		LocationGeo:   location,
		MaxDistanceKm: 50,
	}
	providers, err := s.ProviderRepo.AdvancedSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby providers: %w", err)
	}

	var dtos []models.ProviderDTO
	for i, p := range providers {
		var proximity float64
		coords := p.Profile.LocationGeo.Coordinates
		if len(coords) >= 2 {
			lon, lat := coords[0], coords[1]
			proximity = haversine(location.Coordinates[1], location.Coordinates[0], lat, lon) * 1000
		}
		dtos = append(dtos, models.ProviderDTO{
			ID:               p.ID,
			Profile:          p.Profile,
			ServiceCatalogue: p.ServiceCatalogue,
			LocationGeo:      p.Profile.LocationGeo,
			Preferred:        i == 0,
			Proximity:        proximity,
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
