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

func Update(collectionName, docID string, path string, value interface{}) error {
	if !initialized {
		return fmt.Errorf("firestore client not initialized")
	}

	docRef := firestoreClient.Collection(collectionName).Doc(docID)

	_, err := docRef.Update(ctx, []firestore.Update{{Path: path, Value: value}})
	if err != nil {
		return fmt.Errorf("error during document update: %v", err)
	}

	return nil
}

func Get[T any](collectionName, docID string) (*T, error) {
	data, err := firestoreClient.Collection(collectionName).Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var el T
	data.DataTo(&el)

	return &el, nil
}

func Set(collectionName, docID string, data interface{}) error {
	if !initialized {
		return fmt.Errorf("firestore client not initialized")
	}

	_, err := firestoreClient.Collection(collectionName).Doc(docID).Set(ctx, data)
	if err != nil {
		return fmt.Errorf("error during document set: %v", err)
	}

	return nil
}
