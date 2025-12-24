package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/results/get"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/firestore"
)

func ProcessSessionResults(fc *firestore.FirestoreClient, msgData *bus.ApiResponse, ctx context.Context, pub *pubsub.Publisher) error {
	var err error

	body := []byte(msgData.Body)

	// Convert to the IRacing's API response
	var session get.ResultsGetResponse
	err = json.Unmarshal(body, &session)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	// Convert to a map for Firestore
	// NOTE: this is needed to keep the key names as in the original response
	var sessionMapData map[string]interface{}
	err = json.Unmarshal(body, &sessionMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body to map: %w", err)
	}

	subsessionID := session.SubsessionID
	log.Printf("Processing results for subsession ID: %d", subsessionID)

	// Save to firestore
	err = firestore.UpsertData(fc, "sessions", fmt.Sprintf("%d", subsessionID), map[string]interface{}{
		"spec": map[string]interface{}{
			"session": sessionMapData,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert session results to Firestore: %w", err)
	}

	log.Printf("Successfully saved results for subsession ID: %d", subsessionID)

	// Send request to parse lap data
	var pubsubRequests [][]byte
	var pubsubResults []*pubsub.PublishResult

	for _, simsession := range session.SessionResults {
		for _, simsessionResult := range simsession.Results {
			apiRequest := bus.ApiRequest{
				Endpoint: "/data/results/lap_data",
				Params: map[string]string{
					"subsession_id":     fmt.Sprintf("%d", subsessionID),
					"simsession_number": fmt.Sprintf("%d", simsession.SimsessionNumber),
					"cust_id":           fmt.Sprintf("%d", simsessionResult.CustID),
				},
				Chunks: true,
			}

			data, err := json.Marshal(apiRequest)
			if err != nil {
				log.Printf("Failed to marshal lap data request: %v", err)
				continue
			}

			pubsubRequests = append(pubsubRequests, data)
		}
	}

	for _, reqData := range pubsubRequests {
		result := pub.Publish(ctx, &pubsub.Message{
			Data: reqData,
		})
		pubsubResults = append(pubsubResults, result)
	}

	// Check results
	for _, result := range pubsubResults {
		_, err := result.Get(ctx)
		if err != nil {
			log.Printf("Failed to publish lap data request message: %v", err)
		}
	}

	log.Printf("Published %d lap data requests for subsession ID: %d", len(pubsubRequests), subsessionID)
	return nil
}
