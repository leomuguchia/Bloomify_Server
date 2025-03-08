package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func main() {
	// Initialize the database using the hardcoded URI.
	InitDB()
	coll := MongoClient.Database("bloomify").Collection("providers")
	ctx := context.Background()

	// Filter providers that do not have the "profile" field.
	filter := bson.M{"profile": bson.M{"$exists": false}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to find providers: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var prov bson.M
		if err := cursor.Decode(&prov); err != nil {
			log.Printf("Failed to decode provider: %v", err)
			continue
		}

		profile := bson.M{
			"provider_name":     prov["provider_name"],
			"email":             prov["email"],
			"phone_number":      prov["phone_number"],
			"status":            prov["status"],
			"advanced_verified": prov["advanced_verified"],
			"profile_image":     "", // Leave empty if no value.
		}

		update := bson.M{
			"$set": bson.M{"profile": profile},
		}
		_, err := coll.UpdateOne(ctx, bson.M{"id": prov["id"]}, update)
		if err != nil {
			log.Printf("Failed to update provider %v: %v", prov["id"], err)
		} else {
			log.Printf("Provider %v updated successfully", prov["id"])
		}
	}

	if err := cursor.Err(); err != nil {
		log.Fatalf("Cursor error: %v", err)
	}

	log.Println("Migration complete.")
}
