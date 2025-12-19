package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type FirestoreClient struct {
	client *firestore.Client
	ctx    context.Context
}

func Init(ctx context.Context, projectID string) (*FirestoreClient, error) {
	firestoreClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &FirestoreClient{
		client: firestoreClient,
		ctx:    ctx,
	}, nil
}

// UpsertData writes data to Firestore with upsert behavior.
func UpsertData(fc *FirestoreClient, collectionName, docID string, data interface{}) error {
	docRef := fc.client.Collection(collectionName).Doc(docID)

	_, err := docRef.Set(fc.ctx, data, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("error during document upsert: %v", err)
	}

	return nil
}

func Update(fc *FirestoreClient, collectionName, docID string, path string, value interface{}) error {
	docRef := fc.client.Collection(collectionName).Doc(docID)

	_, err := docRef.Update(fc.ctx, []firestore.Update{{Path: path, Value: value}})
	if err != nil {
		return fmt.Errorf("error during document update: %v", err)
	}

	return nil
}

func Get[T any](fc *FirestoreClient, collectionName, docID string) (*T, error) {
	data, err := fc.client.Collection(collectionName).Doc(docID).Get(fc.ctx)
	if err != nil {
		return nil, err
	}

	var el T
	data.DataTo(&el)

	return &el, nil
}

func Set(fc *FirestoreClient, collectionName, docID string, data interface{}) error {
	_, err := fc.client.Collection(collectionName).Doc(docID).Set(fc.ctx, data)
	if err != nil {
		return fmt.Errorf("error during document set: %v", err)
	}

	return nil
}
