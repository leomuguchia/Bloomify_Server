package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"bloomify/database"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Initialize the database connection.
	database.InitDB()
	client := database.MongoClient
	db := client.Database("bloomify")
	providerColl := db.Collection("providers")

	// Clear existing providers.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := providerColl.DeleteMany(ctx, bson.M{}); err != nil {
		log.Fatalf("Failed to clear providers collection: %v", err)
	}

	// Fixed user point for simulation (Bangalore).
	userLon, userLat := 77.5946, 12.9716

	// Simulation parameters.
	serviceTypes := []string{"cleaning", "laundry", "chauffeur"}
	providersPerService := 10
	totalProviders := len(serviceTypes) * providersPerService

	// Custom options: always "standard": 1.0 plus an extra option.
	extraOptions := []struct {
		Key        string
		Multiplier float64
	}{
		{"luxury", 1.2},
		{"eco", 1.1},
	}

	// Generate dates for the next 7 days.
	var weekDates []string
	today := time.Now()
	for i := 0; i < 7; i++ {
		weekDates = append(weekDates, today.AddDate(0, 0, i).Format("2006-01-02"))
	}

	var providers []interface{}
	rand.Seed(time.Now().UnixNano())
	providerCounter := 1

	// We'll linearly assign distances so that the furthest provider is at 5 km and the closest at ~0.01 km.
	maxDistance := 5.0
	minDistance := 0.01
	spacing := (maxDistance - minDistance) / float64(totalProviders-1)

	// For each service type.
	for _, service := range serviceTypes {
		for i := 1; i <= providersPerService; i++ {
			// Global index for linear distance (0-based).
			globalIndex := float64(providerCounter - 1)
			distanceKm := maxDistance - spacing*globalIndex

			// Random angle (0 to 2π) for positioning within the circle.
			angle := rand.Float64() * 2 * math.Pi

			// Convert radial distance to degree offsets.
			// Approximate: 1 km ≈ 0.00922° longitude and 1 km ≈ 0.009° latitude at this latitude.
			deltaLon := distanceKm * 0.00922 * math.Cos(angle)
			deltaLat := distanceKm * 0.009 * math.Sin(angle)

			locationGeo := models.GeoPoint{
				Type:        "Point",
				Coordinates: []float64{userLon + deltaLon, userLat + deltaLat},
			}

			// Determine mode: first half "provider-to-user", second half "drop-off".
			var mode string
			if i <= providersPerService/2 {
				mode = "provider-to-user"
			} else {
				mode = "drop-off"
			}

			// Randomly choose extra custom option.
			extra := extraOptions[rand.Intn(len(extraOptions))]
			customOptions := map[string]float64{
				"standard": 1.0,
				extra.Key:  extra.Multiplier,
			}

			// Build provider profile.
			profile := models.Profile{
				ProviderName: fmt.Sprintf("%s Provider %d", service, providerCounter),
				Email:        fmt.Sprintf("%s_provider_%d@example.com", service, providerCounter),
				PhoneNumber:  fmt.Sprintf("900000%04d", providerCounter),
				Address:      "123 Sample Street, Sample City",
			}

			// Build service catalogue.
			// Note: Ensure serviceType and mode values match those expected by your matching logic.
			serviceCatalogue := models.ServiceCatalogue{
				ServiceType:   service,
				Mode:          mode,
				CustomOptions: customOptions,
			}

			// Generate weekly timeslots.
			var timeSlots []models.TimeSlot
			for _, dateStr := range weekDates {
				// Earlybird timeslot uses the extra option.
				tsEarly := models.TimeSlot{
					ID:        fmt.Sprintf("ts-%d-%s-early", providerCounter, dateStr),
					Start:     480,  // 8:00 AM
					End:       1020, // 5:00 PM
					Capacity:  30,
					SlotModel: "earlybird",
					UnitType:  "child",
					Date:      dateStr,
					EarlyBird: &models.EarlyBirdSlotData{
						BasePrice:             10.0,
						EarlyBirdDiscountRate: 0.25,
						LateSurchargeRate:     0.10,
					},
					BookedUnitsStandard: 0,
					BookedUnitsPriority: 0,
					Version:             1,
					CustomOptionKey:     extra.Key, // use the extra option key for earlybird
					Mode:                mode,
				}
				// Urgency timeslot uses the "standard" option.
				tsUrgency := models.TimeSlot{
					ID:        fmt.Sprintf("ts-%d-%s-urgency", providerCounter, dateStr),
					Start:     1020, // 5:00 PM
					End:       1380, // 11:00 PM
					Capacity:  20,
					SlotModel: "urgency",
					UnitType:  "child",
					Date:      dateStr,
					Urgency: &models.UrgencySlotData{
						BasePrice:             15.0,
						PrioritySurchargeRate: 0.50,
						ReservedPriority:      5,
						PriorityActive:        true,
					},
					BookedUnitsStandard: 0,
					BookedUnitsPriority: 0,
					Version:             1,
					CustomOptionKey:     "standard",
					Mode:                mode,
				}
				timeSlots = append(timeSlots, tsEarly, tsUrgency)
			}

			// Simulate password hashing.
			pass := "$Password1234"
			hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("failed to hash password: %v", err)
			}

			// Assemble the provider document following registration standards.
			provider := models.Provider{
				ID:                     fmt.Sprintf("prov-%d", providerCounter),
				Profile:                profile,
				LegalName:              profile.ProviderName,
				Password:               "", // Do not store raw password
				PasswordHash:           string(hashed),
				ServiceCatalogue:       serviceCatalogue,
				Location:               "Sample City",
				LocationGeo:            locationGeo,
				Rating:                 0,
				CompletedBookings:      0,
				TimeSlots:              timeSlots,
				AcceptedPaymentMethods: []string{"inApp", "cash"},
				PrePaymentRequired:     false,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
				Devices:                []models.Device{}, // Assume no devices at registration time
			}

			providers = append(providers, provider)
			providerCounter++
		}
	}

	// Insert all providers into MongoDB.
	insertResult, err := providerColl.InsertMany(ctx, providers)
	if err != nil {
		log.Fatalf("Failed to insert providers: %v", err)
	}
	fmt.Printf("Inserted provider IDs: %v\n", insertResult.InsertedIDs)
}
