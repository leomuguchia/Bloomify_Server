package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"bloomify/database/repository"
	"bloomify/models"

	"github.com/go-redis/redis/v8"
)

// MatchingService defines methods to match providers based on a service plan.
type MatchingService interface {
	MatchProviders(plan models.ServicePlan) ([]models.Provider, error)
}

// DefaultMatchingService is our robust implementation.
type DefaultMatchingService struct {
	ProviderRepo repository.ProviderRepository
	CacheClient  *redis.Client
}

// MatchProviders retrieves a ranked list of providers matching the given service plan.
// It first attempts to retrieve the result from cache; if not found, it computes the match and caches it.
func (s *DefaultMatchingService) MatchProviders(plan models.ServicePlan) ([]models.Provider, error) {
	ctx := context.Background()

	// Create a cache key based on the JSON representation of the plan.
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service plan: %w", err)
	}
	cacheKey := fmt.Sprintf("match:%x", planBytes)

	// Try to get from cache.
	cached, err := s.CacheClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var providers []models.Provider
		if err := json.Unmarshal([]byte(cached), &providers); err == nil {
			return providers, nil
		}
		// If unmarshal fails, we fall through to re-computation.
	}

	// Retrieve providers matching service type.
	allProviders, err := s.ProviderRepo.GetByServiceType(plan.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve providers: %w", err)
	}
	if len(allProviders) == 0 {
		return nil, fmt.Errorf("no providers found for service '%s'", plan.Service)
	}

	// Scoring constants.
	const (
		BaseLocationScore     = 100.0
		DistancePenalty       = 2.0
		UrgencyHighMultiplier = 1.5
		UrgencyLowMultiplier  = 1.1
		CapacityThreshold     = 3
		RatingWeight          = 10.0
		BookingWeight         = 5.0
		LocationWeight        = 0.4
		CapacityWeight        = 0.3
		HistoryWeight         = 0.3
	)

	type scoredProvider struct {
		Provider models.Provider
		Score    float64
	}
	var scored []scoredProvider

	for _, p := range allProviders {
		// For location filtering, we assume the provider must be in the same region.
		if !strings.EqualFold(p.Location, plan.Location) {
			continue
		}

		// Compute distance using Haversine.
		distanceKm := haversine(plan.Latitude, plan.Longitude, p.Latitude, p.Longitude)
		locationScore := BaseLocationScore - (distanceKm * DistancePenalty)
		if locationScore < 0 {
			locationScore = 0
		}

		// Capacity score with urgency adjustment.
		capacityScore := float64(p.Capacity)
		if strings.EqualFold(plan.Urgency, "Now") {
			if p.Capacity >= CapacityThreshold {
				capacityScore *= UrgencyHighMultiplier
			} else {
				capacityScore *= UrgencyLowMultiplier
			}
		}

		// Historical performance score.
		historicalScore := (p.Rating * RatingWeight) + (math.Log(float64(p.CompletedBookings)+1) * BookingWeight)

		finalScore := (LocationWeight * locationScore) + (CapacityWeight * capacityScore) + (HistoryWeight * historicalScore)

		scored = append(scored, scoredProvider{Provider: p, Score: finalScore})
	}

	if len(scored) == 0 {
		return nil, fmt.Errorf("no providers found matching service '%s' near your location", plan.Service)
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	var matched []models.Provider
	for _, sp := range scored {
		matched = append(matched, sp.Provider)
	}

	// Cache the result for 5 minutes.
	matchedBytes, err := json.Marshal(matched)
	if err == nil {
		s.CacheClient.Set(ctx, cacheKey, matchedBytes, 5*time.Minute)
	}

	return matched, nil
}

// haversine calculates the great-circle distance (in km) between two lat/lon points.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
