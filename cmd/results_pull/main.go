package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/pubsub/v2"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/processing"
)

const (
	responseSubscriptionID = "sub-api-res"
	requestTopicID         = "api-req"
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
	sub := pubSubClient.Subscriber(responseSubscriptionID)
	pub := pubSubClient.Publisher(requestTopicID)

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

		switch msgData.Endpoint {
		case "/data/results/get":
			err = processing.ProcessSessionResults(&msgData, ctx, pub)
			if err != nil {
				log.Printf("Failed to process session results: %v", err)
				msg.Nack()
				return
			}

		case "/data/results/lap_data":
			err = processing.ProcessSessionLaps(&msgData)
			if err != nil {
				log.Printf("Failed to process session laps: %v", err)
				msg.Nack()
				return
			}

		case "/data/league/season_sessions":
			err = processing.ProcessLeagueSeasonSessions(&msgData, ctx, pub)
			if err != nil {
				log.Printf("Failed to process league season sessions: %v", err)
				msg.Nack()
				return
			}

		default:
			log.Printf("Skipping unknown endpoint: %s", msgData.Endpoint)
			msg.Ack()
			return
		}

		// Acknowledge the message
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("sub.Receive: %v", err)
	}
}
