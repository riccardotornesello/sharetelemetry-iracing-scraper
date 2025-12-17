package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/results/get"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/firestore"
)

const (
	responseSubscriptionID = "sub-api-res"
)

func main() {
	ctx := context.Background()

	projectID := os.Getenv("PROJECT_ID")

	// Create a Pub/Sub client
	pubSubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubSubClient.Close()

	// Create subscriber and publisher
	// TODO: filter messages
	// TODO: publish request for lap data
	sub := pubSubClient.Subscriber(responseSubscriptionID)

	// Parse messages
	log.Println("Listening for messages...")
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		var msgData bus.ApiResponse
		err := json.Unmarshal(msg.Data, &msgData)
		if err != nil {
			log.Printf("Failed to unmarshal message data: %v", err)
			msg.Nack()
			return
		}

		// Convert to the IRacing's API response
		var results get.ResultsGetResponse
		err = json.Unmarshal([]byte(msgData.Body), &results)
		if err != nil {
			log.Printf("Failed to unmarshal API response body: %v", err)
			msg.Nack()
			return
		}

		// Convert to a map for Firestore
		// NOTE: this is needed to keep the key names as in the original response
		var resultsMapData map[string]interface{}
		err = json.Unmarshal([]byte(msgData.Body), &resultsMapData)
		if err != nil {
			log.Printf("Failed to unmarshal API response body to map: %v", err)
			msg.Nack()
			return
		}

		subsessionID := results.SubsessionID
		log.Printf("Processing results for subsession ID: %d", subsessionID)

		// Save to firestore
		err = firestore.UpsertData("sessions", fmt.Sprintf("%d", subsessionID), map[string]interface{}{
			"spec": resultsMapData,
		})
		if err != nil {
			log.Printf("Failed to upsert data to Firestore: %v", err)
			msg.Nack()
			return
		}

		log.Printf("Successfully saved results for subsession ID: %d", subsessionID)

		// Acknowledge the message
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("sub.Receive: %v", err)
	}
}
