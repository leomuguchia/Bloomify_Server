// File: migrate_provider_timeslots.go
package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	providerRepo "bloomify/database/repository/provider"
	"bloomify/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// We'll use the previous DB setup code:
var MongoClient *mongo.Client

// InitDB initializes the MongoDB connection with a hardcoded URI.
func InitDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("failed to ping MongoDB: %v", err)
	}
	MongoClient = client
	log.Println("Connected to MongoDB successfully!")
}

// generateTimeslots creates two timeslots per day for the next 'days' days based on provider type.
// For individuals, capacity is fixed at 1; for businesses, capacity is randomized between minCap and maxCap.
func generateTimeslots(providerType string, days int, minCap, maxCap int) []models.TimeSlot {
	var timeslots []models.TimeSlot
	now := time.Now()
	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, i)
		dateStr := day.Format("2006-01-02")

		var capacityMorning, capacityAfternoon int
		if providerType == "business" {
			capacityMorning = rand.Intn(maxCap-minCap+1) + minCap
			capacityAfternoon = rand.Intn(maxCap-minCap+1) + minCap
		} else {
			capacityMorning = 1
			capacityAfternoon = 1
		}

		// Morning timeslot: 7:00 AM (420 minutes) to 12:00 PM (720 minutes)
		morning := models.TimeSlot{
			Start:     420,
			End:       720,
			Capacity:  capacityMorning,
			SlotModel: "flatrate",
			UnitType:  "child", // adjust as needed
			Date:      dateStr,
			Flatrate:  &models.FlatrateSlotData{BasePrice: 25.0},
			Version:   1,
		}

		// Afternoon timeslot: 2:00 PM (840 minutes) to 5:00 PM (1020 minutes)
		afternoon := models.TimeSlot{
			Start:     840,
			End:       1020,
			Capacity:  capacityAfternoon,
			SlotModel: "flatrate",
			UnitType:  "child", // adjust as needed
			Date:      dateStr,
			Flatrate:  &models.FlatrateSlotData{BasePrice: 20.0},
			Version:   1,
		}

		timeslots = append(timeslots, morning, afternoon)
	}
	return timeslots
}

func main() {
	// Initialize the database connection.
	InitDB()
	_, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get the providers repository.
	repo := providerRepo.NewMongoProviderRepo()

	// Fetch all providers.
	providers, err := repo.GetAll()
	if err != nil {
		log.Fatalf("Failed to fetch providers: %v", err)
	}
	log.Printf("Found %d providers", len(providers))

	// Iterate over each provider.
	for _, prov := range providers {
		// Randomly assign provider type if not already set.
		if prov.ProviderType == "" {
			if rand.Float32() < 0.5 {
				prov.ProviderType = "individual"
			} else {
				prov.ProviderType = "business"
			}
		}

		// Choose capacity range for business providers.
		minCapacity := 10
		maxCapacity := 50

		// Generate timeslots for 7 days.
		ts := generateTimeslots(prov.ProviderType, 7, minCapacity, maxCapacity)
		prov.TimeSlots = ts

		// Update provider status to active if timeslots are set.
		// For this migration, we'll update the status under the profile.
		prov.Profile.Status = "active"

		// Update the provider document.
		if err := repo.Update(&prov); err != nil {
			log.Printf("Failed to update provider %s: %v", prov.ID, err)
		} else {
			log.Printf("Provider %s updated with provider_type '%s' and %d timeslots", prov.ID, prov.ProviderType, len(ts))
		}
	}

	log.Println("Provider timeslot migration complete.")
}
