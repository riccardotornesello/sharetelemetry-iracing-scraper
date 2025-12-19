package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/pubsub/v2"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/iracing"
)

const (
	requestSubscriptionID = "sub-api-req"
	responseTopicID       = "api-res"
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
	sub := pubSubClient.Subscriber(requestSubscriptionID)
	pub := pubSubClient.Publisher(responseTopicID)

	// Parse messages
	log.Println("Listening for messages...")
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		var msgData bus.ApiRequest
		err := json.Unmarshal(msg.Data, &msgData)
		if err != nil {
			log.Printf("Failed to unmarshal message data: %v", err)
			msg.Nack()
			return
		}

		err = iracing.HandleApiRequest(ctx, pub, &msgData)
		if err != nil {
			log.Printf("Failed to handle API request: %v", err)
			msg.Nack()
			return
		}

		// Acknowledge the message
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("sub.Receive: %v", err)
	}
}
