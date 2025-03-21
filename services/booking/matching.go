package booking

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"bloomify/database/repository"
	"bloomify/models"
)

type RankedProvider struct {
	Provider   models.Provider
	RankPoints float64
	Preferred  bool
}

type MatchingService interface {
	MatchProviders(plan models.ServicePlan) ([]models.ProviderDTO, error)
}

type DefaultMatchingService struct {
	ProviderRepo repository.ProviderRepository
}

func (s *DefaultMatchingService) MatchProviders(plan models.ServicePlan) ([]models.ProviderDTO, error) {
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
	return extractProvidersDTO(rankedProviders), nil
}

func (s *DefaultMatchingService) matchProviders(criteria repository.ProviderSearchCriteria, ctx context.Context) ([]RankedProvider, error) {
	providers, err := s.ProviderRepo.AdvancedSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("advanced search failed: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found for service '%s'", criteria.ServiceType)
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

	type scoreData struct {
		Provider       models.Provider
		TotalScore     float64
		LocationScore  float64
		VerifiedScore  float64
		CompletedScore float64
		RatingScore    float64
	}

	resultsCh := make(chan scoreData, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)
		go func(p models.Provider) {
			defer wg.Done()
			var provLat, provLon float64
			if len(p.LocationGeo.Coordinates) >= 2 {
				provLon = p.LocationGeo.Coordinates[0]
				provLat = p.LocationGeo.Coordinates[1]
			}
			distanceKm := haversine(centerLat, centerLon, provLat, provLon)
			locScore := computeLocationScore(distanceKm)
			var verifiedScore float64
			if p.Profile.AdvancedVerified {
				verifiedScore = VerifiedBonus
			}
			compScore := computeCompletedScore(p.CompletedBookings)
			ratingScore := computeRatingScore(p.Rating)
			totalScore := locScore + verifiedScore + compScore + ratingScore

			resultsCh <- scoreData{
				Provider:       p,
				TotalScore:     totalScore,
				LocationScore:  locScore,
				VerifiedScore:  verifiedScore,
				CompletedScore: compScore,
				RatingScore:    ratingScore,
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

func extractProvidersDTO(ranked []RankedProvider) []models.ProviderDTO {
	var dtos []models.ProviderDTO
	for _, rp := range ranked {
		dto := models.ProviderDTO{
			ID:               rp.Provider.ID,
			Profile:          rp.Provider.Profile,
			ServiceCatalogue: rp.Provider.ServiceCatalogue,
			LocationGeo:      rp.Provider.LocationGeo,
			Preferred:        rp.Preferred,
		}
		dtos = append(dtos, dto)
	}
	return dtos
}
