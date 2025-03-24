package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"bloomify/config"
	"bloomify/database"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Ensure configuration is loaded so that DATABASE_URL is set.
	config.LoadConfig()
	// Optionally, force the environment variable if config isn't loading in tests:
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "mongodb://localhost:27017")
		config.LoadConfig()
	}

	// Initialize the database connection.
	database.InitDB()
	client := database.MongoClient
	db := client.Database("bloomify")
	providerColl := db.Collection("providers")

	// Clear existing providers for a clean simulation.
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

	// Custom options: every provider always has "standard":1.0, plus one extra option.
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

	// Loop over each service type.
	for _, service := range serviceTypes {
		for i := 1; i <= providersPerService; i++ {
			// Global index (0-based) for distance.
			globalIndex := float64(providerCounter - 1)
			distanceKm := maxDistance - spacing*globalIndex

			// Random angle (0 to 2π) to distribute providers around the user.
			angle := rand.Float64() * 2 * math.Pi

			// Convert radial distance (km) to degree offsets.
			// At Bangalore, approximately 1 km ≈ 0.00922° longitude and ≈ 0.009° latitude.
			deltaLon := distanceKm * 0.00922 * math.Cos(angle)
			deltaLat := distanceKm * 0.009 * math.Sin(angle)

			// Geo-location stored inside the provider's Profile.
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

			// Randomly choose an extra custom option.
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
				Status:       "active",
				ProfileImage: "https://example.com/default_profile.png",
				LocationGeo:  locationGeo,
			}

			// Build service catalogue.
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
					CustomOptionKey:     extra.Key, // extra option for earlybird
					Mode:                mode,
				}
				// Urgency timeslot uses "standard".
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

			// Hash the provider password.
			rawPassword := "$Password1234"
			hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}

			// Assemble the provider document using the updated models.
			provider := models.Provider{
				ID:      fmt.Sprintf("prov-%d", providerCounter),
				Profile: profile,
				// Security details are now encapsulated in the Security struct.
				Security: models.Security{
					PasswordHash: string(hashed),
				},
				ServiceCatalogue: serviceCatalogue,
				// Basic verification details are moved into the BasicVerification struct.
				BasicVerification: models.BasicVerification{
					LegalName:          profile.ProviderName,
					KYPDocument:        "",
					VerificationStatus: "unverified",
				},
				VerificationLevel:    "",                            // e.g., "basic" or "advanced" as needed.
				AdvancedVerification: models.AdvancedVerification{}, // Empty for now.
				HistoricalRecords:    nil,
				TimeSlots:            timeSlots,
				PaymentDetails: models.PaymentDetails{
					AcceptedPaymentMethods: []string{"inApp", "cash"},
					PrePaymentRequired:     false,
				},
				CompletedBookings: 0,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
				Devices:           []models.Device{},
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
