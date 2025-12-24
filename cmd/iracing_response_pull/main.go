package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/pubsub/v2"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/database"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/processing"
)

const (
	responseSubscriptionID = "sub-api-res"
	requestTopicID         = "api-req"
)

func main() {
	ctx := context.Background()

	projectID := os.Getenv("PROJECT_ID")

	dbUri := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("MONGODB_DATABASE")

	// Create a Pub/Sub client
	pubSubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubSubClient.Close()

	// Create subscriber and publisher
	sub := pubSubClient.Subscriber(responseSubscriptionID)
	pub := pubSubClient.Publisher(requestTopicID)

	// Connect to the database
	db := database.Connect(dbUri, dbName)

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

		err = processing.MultiplexProcessing(db, ctx, pub, &msgData)
		if err != nil {
			log.Printf("Failed to process message: %v", err)
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
