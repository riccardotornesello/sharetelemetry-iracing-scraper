package sharetelemetryiracingscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/pubsub/v2"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/iracing"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/processing"
)

type MessagePublishedData struct {
	Message struct {
		Data []byte `json:"data"`
	} `json:"message"`
}

var (
	projectID = os.Getenv("PROJECT_ID")

	apiRequestTopicID  = os.Getenv("API_REQUEST_TOPIC_ID")
	apiResponseTopicID = os.Getenv("API_RESPONSE_TOPIC_ID")

	pubSubClient *pubsub.Client
)

func init() {
	var err error

	pubSubClient, err = pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		panic(fmt.Sprintf("pubsub.NewClient: %v", err))
	}

	functions.CloudEvent("ApiPull", apiPull)
	functions.CloudEvent("ResponsePull", responsePull)
}

func apiPull(ctx context.Context, e event.Event) error {
	var msg MessagePublishedData
	if err := e.DataAs(&msg); err != nil {
		return fmt.Errorf("event.DataAs: %w", err)
	}

	var msgData bus.ApiRequest
	err := json.Unmarshal(msg.Message.Data, &msgData)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	pub := pubSubClient.Publisher(apiResponseTopicID)

	return iracing.HandleApiRequest(ctx, pub, &msgData)
}

func responsePull(ctx context.Context, e event.Event) error {
	var msg MessagePublishedData
	if err := e.DataAs(&msg); err != nil {
		return fmt.Errorf("event.DataAs: %w", err)
	}

	var msgData bus.ApiResponse
	err := json.Unmarshal(msg.Message.Data, &msgData)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	pub := pubSubClient.Publisher(apiRequestTopicID)

	return processing.MultiplexProcessing(ctx, pub, &msgData)
}
