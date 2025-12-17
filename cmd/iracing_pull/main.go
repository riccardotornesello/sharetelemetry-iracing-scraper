package main

import (
	"context"
	"encoding/json"
	"io"
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

		res, err := iracing.CallApi(msgData.Endpoint, msgData.Params)
		if err != nil {
			// TODO: handle error response properly
			log.Printf("API call failed: %v", err)
			msg.Nack()
			return
		}
		defer res.Body.Close()

		log.Printf("API call to '%s' succeeded with status: %s", msgData.Endpoint, res.Status)

		// Publish the response body to the response topic
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			msg.Nack()
			return
		}

		apiResponse := bus.ApiResponse{
			Endpoint: msgData.Endpoint,
			Params:   msgData.Params,
			Body:     string(bodyBytes),
		}

		data, err := json.Marshal(apiResponse)
		if err != nil {
			log.Printf("Failed to marshal response data: %v", err)
			msg.Nack()
			return
		}

		result := pub.Publish(ctx, &pubsub.Message{
			Data: data,
			Attributes: map[string]string{
				"endpoint": msgData.Endpoint,
			},
		})
		_, err = result.Get(ctx)
		if err != nil {
			log.Printf("Failed to publish response message: %v", err)
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
