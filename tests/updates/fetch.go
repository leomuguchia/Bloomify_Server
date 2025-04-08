package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Provider represents a simplified structure for the provider document.
type Provider struct {
	ID        string      `bson:"id"`
	TimeSlots interface{} `bson:"timeSlots"`
}

func Test() {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB. Adjust the URI as necessary.
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Select the "providers" collection in the "bloomify" database.
	collection := client.Database("bloomify").Collection("providers")

	// Fetch all provider documents.
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Error fetching providers: %v", err)
	}
	defer cursor.Close(ctx)

	// Iterate through the cursor and print each provider's structure.
	for cursor.Next(ctx) {
		var provider Provider
		if err := cursor.Decode(&provider); err != nil {
			log.Printf("Error decoding provider: %v", err)
			continue
		}
		fmt.Printf("Provider ID: %s\n", provider.ID)
		fmt.Printf("TimeSlots: %+v\n\n", provider.TimeSlots)
	}
	if err := cursor.Err(); err != nil {
		log.Fatalf("Cursor error: %v", err)
	}
}
