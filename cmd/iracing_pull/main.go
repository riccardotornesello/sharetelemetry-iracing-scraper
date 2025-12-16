package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/pubsub/v2"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/iracing"
)

const (
	projectID             = "demo-sharetelemetry"
	requestSubscriptionID = "sub-api-req"
)

type ApiRequest struct {
	Endpoint string                 `json:"endpoint"`
	Params   map[string]interface{} `json:"params"`
}

func main() {
	ctx := context.Background()

	// Connect to the Pub/Sub emulator
	err := os.Setenv("PUBSUB_EMULATOR_HOST", "127.0.0.1:8085")
	if err != nil {
		log.Fatalf("Failed to set PUBSUB_EMULATOR_HOST: %v", err)
	}

	// Create a Pub/Sub client
	pubSubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubSubClient.Close()

	// Subscribe to the subscription to receive messages
	sub := pubSubClient.Subscriber(requestSubscriptionID)

	// Parse messages
	log.Println("Listening for messages...")
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		var msgData ApiRequest
		err := json.Unmarshal(msg.Data, &msgData)
		if err != nil {
			log.Printf("Failed to unmarshal message data: %v", err)
			msg.Nack()
			return
		}

		res, err := iracing.CallApi(msgData.Endpoint, msgData.Params)
		if err != nil {
			log.Printf("API call failed: %v", err)
			msg.Nack()
			return
		}
		defer res.Body.Close()

		log.Printf("API call to '%s' succeeded with status: %s", msgData.Endpoint, res.Status)

		// TODO: publish the response to another topic

		// Acknowledge the message
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("sub.Receive: %v", err)
	}
}
