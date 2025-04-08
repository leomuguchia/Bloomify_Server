package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"bloomify/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// randomInt returns a random integer between min and max (inclusive).
func randomInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// CandidateSlot defines a realistic time range (in minutes from midnight) for a slot.
type CandidateSlot struct {
	Start int // in minutes from midnight
	End   int // in minutes from midnight
}

// Define some realistic candidate time ranges.
var candidateSlots = []CandidateSlot{
	{Start: 480, End: 660},   // 8:00 AM - 11:00 AM
	{Start: 720, End: 840},   // 12:00 PM - 2:00 PM
	{Start: 900, End: 1080},  // 3:00 PM - 6:00 PM
	{Start: 1140, End: 1260}, // 7:00 PM - 9:00 PM
}

// generateRandomTimeslot creates a realistic timeslot for the given date.
// It picks one of the candidate slots and then may adjust the times slightly.
func generateRandomTimeslot(date string, capacity int) models.TimeSlot {
	// Choose one candidate slot at random.
	cand := candidateSlots[rand.Intn(len(candidateSlots))]
	// Slightly randomize start and end within 10 minutes.
	startAdjustment := randomInt(-5, 5)
	endAdjustment := randomInt(-5, 5)
	start := cand.Start + startAdjustment
	end := cand.End + endAdjustment
	if end <= start {
		end = start + 30 // ensure minimum duration of 30 minutes
	}
	// Randomly choose a slot model.
	slotModels := []string{"urgency", "earlybird", "flatrate"}
	slotModel := slotModels[rand.Intn(len(slotModels))]

	// Instead of leaving unitType empty, select a service and use its UnitType.
	services := []models.Service{
		{ID: "1", Name: "Babysitting", Icon: "baby-buggy", UnitType: "kids", ProviderTerm: "Babysitters"},
		{ID: "2", Name: "Chauffeuring", Icon: "steering", UnitType: "hour", ProviderTerm: "Chauffeurs"},
		{ID: "3", Name: "Laundry", Icon: "washing-machine", UnitType: "kg", ProviderTerm: "Laundry Service"},
		{ID: "4", Name: "Cleaning", Icon: "broom", UnitType: "hour", ProviderTerm: "Cleaning Professionals"},
		{ID: "5", Name: "Plumbing", Icon: "pipe-wrench", UnitType: "hour", ProviderTerm: "Plumbers"},
		{ID: "6", Name: "Electrical", Icon: "flash", UnitType: "hour", ProviderTerm: "Electricians"},
		{ID: "7", Name: "Delivery", Icon: "truck-delivery", UnitType: "kg", ProviderTerm: "Delivery Personnel"},
		{ID: "8", Name: "Pet Sitting", Icon: "paw", UnitType: "hour", ProviderTerm: "Pet Sitters"},
		{ID: "9", Name: "Tutoring", Icon: "book", UnitType: "hour", ProviderTerm: "Tutors"},
		{ID: "10", Name: "Fitness Training", Icon: "dumbbell", UnitType: "hour", ProviderTerm: "Trainers"},
	}
	selectedService := services[rand.Intn(len(services))]
	unitType := selectedService.UnitType

	// Create the base timeslot.
	ts := models.TimeSlot{
		ID:                  fmt.Sprintf("%d", time.Now().UnixNano()), // simple ID using timestamp
		Start:               start,
		End:                 end,
		Capacity:            capacity,
		SlotModel:           slotModel,
		UnitType:            unitType, // now set correctly from the service definition
		Date:                date,
		Version:             1,
		BookedUnitsStandard: 0,
		BookedUnitsPriority: 0,
	}

	// Populate model-specific data with plausible simulation values.
	switch slotModel {
	case "urgency":
		var reserved int
		if capacity <= 1 {
			reserved = 0
		} else {
			reserved = randomInt(1, capacity/4)
		}
		ts.Urgency = &models.UrgencySlotData{
			BasePrice:             float64(randomInt(20, 40)),
			PrioritySurchargeRate: float64(randomInt(50, 100)) / 100.0,
			ReservedPriority:      reserved,
			PriorityActive:        true,
		}
	case "earlybird":
		ts.EarlyBird = &models.EarlyBirdSlotData{
			BasePrice:             float64(randomInt(15, 35)), // e.g., 15-35 units
			EarlyBirdDiscountRate: 0.25,                       // 25% discount
			LateSurchargeRate:     0.25,                       // 25% surcharge
		}
	case "flatrate":
		ts.Flatrate = &models.FlatrateSlotData{
			BasePrice: float64(randomInt(25, 50)), // e.g., 25-50 units
		}
	}

	return ts
}

// updateProviderTimeslots updates every provider document in the "providers" collection,
// adding recurring timeslot data. It supports both individual and group providers.
func updateProviderTimeslots(client *mongo.Client, dbName string) error {
	ctx := context.Background()
	providerColl := client.Database(dbName).Collection("providers")

	// Get all provider documents.
	cursor, err := providerColl.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to find providers: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var provider bson.M
		if err := cursor.Decode(&provider); err != nil {
			log.Printf("failed to decode provider: %v", err)
			continue
		}

		// Get provider id.
		providerID, ok := provider["id"].(string)
		if !ok || providerID == "" {
			log.Println("provider missing id, skipping")
			continue
		}

		// Check provider type: if individual, capacity = 1; else, random between 20 and 50.
		capacity := randomInt(20, 50)
		if profile, ok := provider["profile"].(bson.M); ok {
			if providerType, ok := profile["providerType"].(string); ok && providerType == "individual" {
				capacity = 1
			}
		}

		// Choose a recurring period randomly between 1 and 5 weeks.
		recurringWeeks := randomInt(1, 5)
		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, recurringWeeks*7)

		var newSlots []models.TimeSlot
		// Loop over each day from startDate to endDate.
		for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			// Generate between 1 and 4 timeslots for this day.
			numSlots := randomInt(1, 4)
			for i := 0; i < numSlots; i++ {
				ts := generateRandomTimeslot(dateStr, capacity)
				newSlots = append(newSlots, ts)
			}
		}

		// Update provider document: add/update the "timeSlots" field.
		update := bson.M{
			"$set": bson.M{
				"timeSlots": newSlots,
			},
		}
		_, err := providerColl.UpdateOne(ctx, bson.M{"id": providerID}, update)
		if err != nil {
			log.Printf("failed to update provider %s with timeslots: %v", providerID, err)
		} else {
			log.Printf("Provider %s updated with %d timeslots (recurring for %d weeks)", providerID, len(newSlots), recurringWeeks)
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}
	return nil
}

func main() {
	// Seed the random generator.
	rand.Seed(time.Now().UnixNano())

	// Connect to MongoDB.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := "mongodb://localhost:27017" // Change if necessary.
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	dbName := "bloomify" // Change if your database name is different.

	// Update timeslot data for all providers.
	if err := updateProviderTimeslots(client, dbName); err != nil {
		log.Fatalf("Error updating provider timeslots: %v", err)
	}

	log.Println("All providers updated with recurring timeslot data successfully.")
}
