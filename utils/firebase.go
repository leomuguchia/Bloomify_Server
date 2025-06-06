package utils

import (
	"bloomify/config"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var (
	FCMClient     *messaging.Client
	StorageClient *storage.BucketHandle
	BucketName    string
)

// LoadServiceAccount loads service account JSON from a file
func LoadServiceAccount(path string) (*config.ServiceAccount, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}

	var sa config.ServiceAccount
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, fmt.Errorf("failed to parse service account JSON: %w", err)
	}

	// Clean up private key formatting if needed
	sa.PrivateKey = strings.ReplaceAll(sa.PrivateKey, `\n`, "\n")

	return &sa, nil
}

// FirebaseInit initializes Firebase app, messaging client, and storage bucket handle
func FirebaseInit() {
	ctx := context.Background()
	serviceAccountPath := config.FirebaseServiceAccountKeyPath

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("firebase: error initializing app: %v", err)
	}

	// Initialize FCM client
	fcmClient, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("firebase: error getting messaging client: %v", err)
	}
	FCMClient = fcmClient

	// Initialize Storage client and get default bucket
	storageClient, err := app.Storage(ctx)
	if err != nil {
		log.Fatalf("firebase: error initializing storage client: %v", err)
	}

	bucket, err := storageClient.Bucket(config.FirebaseBucketName)
	if err != nil {
		log.Fatalf("firebase: error getting storage bucket: %v", err)
	}

	StorageClient = bucket
	BucketName = bucket.BucketName()
}
