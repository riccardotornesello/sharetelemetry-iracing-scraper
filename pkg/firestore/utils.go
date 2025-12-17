package firestore

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
)

var (
	initialized bool

	ctx             = context.Background()
	firestoreClient *firestore.Client
)

func init() {
	var err error

	projectID := os.Getenv("PROJECT_ID")

	// Initialize Firestore client
	firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		fmt.Printf("Error initializing Firestore client: %v\n", err)
		return
	}

	// Mark as initialized
	initialized = true
}

// UpsertData writes data to Firestore with upsert behavior.
func UpsertData(collectionName, docID string, data interface{}) error {
	if !initialized {
		return fmt.Errorf("firestore client not initialized")
	}

	docRef := firestoreClient.Collection(collectionName).Doc(docID)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("error during document upsert: %v", err)
	}

	return nil
}
