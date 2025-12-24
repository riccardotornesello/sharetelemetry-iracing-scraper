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
	"github.com/riccardotornesello/irapi-go/pkg/client"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/bus"
)

func HandleApiRequest(ctx context.Context, iracingClient *irapi.IRacingApiClient, pub *pubsub.Publisher, msgData *bus.ApiRequest) error {
	var err error
	var chunksData *string

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

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse chunks if requested
	if msgData.Chunks {
		// Get chunk_info from response body
		var bodyMap map[string]json.RawMessage
		err = json.Unmarshal(bodyBytes, &bodyMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body for chunk info: %v", err)
		}

		var chunkInfo client.IRacingChunkInfo
		err = json.Unmarshal(bodyMap["chunk_info"], &chunkInfo)
		if err != nil {
			return fmt.Errorf("failed to unmarshal chunk_info: %v", err)
		}

		// Fetch all chunks
		fullData, err := client.GetChunks[map[string]interface{}](&chunkInfo)
		if err != nil {
			return fmt.Errorf("failed to get chunks: %v", err)
		}

		log.Printf("Successfully retrieved %d chunks for endpoint '%s'", chunkInfo.NumChunks, msgData.Endpoint)

		// Marshal full data back to JSON
		fullDataBytes, err := json.Marshal(fullData)
		if err != nil {
			return fmt.Errorf("failed to marshal full chunked data: %v", err)
		}

		chunksStr := string(fullDataBytes)
		chunksData = &chunksStr
	}

	// Publish the response body to the response topic
	apiResponse := bus.ApiResponse{
		Endpoint: msgData.Endpoint,
		Params:   msgData.Params,
		Body:     string(bodyBytes),
		Chunks:   chunksData,
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
