package sharetelemetryiracingscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/pubsub/v2"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/riccardotornesello/irapi-go"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/bus"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/database"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/iracing"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/processing"
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

	dbUri  = os.Getenv("MONGODB_URI")
	dbName = os.Getenv("MONGODB_DATABASE")

	ctx           = context.Background()
	pubSubClient  *pubsub.Client
	iracingClient *irapi.IRacingApiClient
	db            *database.DB
)

func init() {
	var err error

	// Connect to Pub/Sub
	pubSubClient, err = pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		panic(fmt.Sprintf("pubsub.NewClient: %v", err))
	}

	// Connect to iRacing
	iracingClient, err = irapi.NewIRacingPasswordLimitedApiClient(
		os.Getenv("IRACING_CLIENT_ID"),
		os.Getenv("IRACING_CLIENT_SECRET"),
		os.Getenv("IRACING_USERNAME"),
		os.Getenv("IRACING_PASSWORD"),
	)
	if err != nil {
		panic(fmt.Sprintf("Error initializing iRacing client: %v", err))
	}

	// Connect to the database
	db = database.Connect(dbUri, dbName)

	// Register Cloud Functions
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

	return iracing.HandleApiRequest(ctx, iracingClient, pub, &msgData)
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

	return processing.MultiplexProcessing(db, ctx, pub, &msgData)
}
