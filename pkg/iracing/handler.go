package iracing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
)

func HandleApiRequest(ctx context.Context, iracingClient *irapi.IRacingApiClient, pub *pubsub.Publisher, msgData *bus.ApiRequest) error {
	var err error

	// Generate the query parameters
	paramsValues := url.Values{}
	for k, v := range msgData.Params {
		paramsValues.Add(k, fmt.Sprintf("%v", v))
	}

	res, err := iracingClient.Client.Get(msgData.Endpoint, paramsValues.Encode())
	if err != nil {
		// TODO: handle error response properly
		return fmt.Errorf("API call failed: %v", err)
	}
	defer res.Body.Close()

	log.Printf("API call to '%s' succeeded with status: %s", msgData.Endpoint, res.Status)

	// Publish the response body to the response topic
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	apiResponse := bus.ApiResponse{
		Endpoint: msgData.Endpoint,
		Params:   msgData.Params,
		Body:     string(bodyBytes),
	}

	data, err := json.Marshal(apiResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal API response: %v", err)
	}

	result := pub.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"endpoint": msgData.Endpoint,
		},
	})
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish response message: %v", err)
	}

	return nil
}
