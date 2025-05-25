// utils/firebase.go
package utils

import (
	"bloomify/config"
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var FCMClient *messaging.Client

// Init initializes the Firebase App and Messaging client.
func FirebaseInit() {
	ctx := context.Background()
	opt := option.WithCredentialsFile(config.FirebaseServiceAccountKeyPath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("firebase: error initializing app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("firebase: error getting Messaging client: %v", err)
	}

	FCMClient = client
}
