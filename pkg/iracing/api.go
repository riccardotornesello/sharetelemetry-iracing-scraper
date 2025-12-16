package iracing

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/riccardotornesello/irapi-go"
)

var (
	initialized   bool
	iracingClient *irapi.IRacingApiClient
)

func init() {
	var err error

	// Initialize the iRacing client
	iracingClient, err = irapi.NewIRacingPasswordLimitedApiClient(
		os.Getenv("IRACING_CLIENT_ID"),
		os.Getenv("IRACING_CLIENT_SECRET"),
		os.Getenv("IRACING_USERNAME"),
		os.Getenv("IRACING_PASSWORD"),
	)

	if err != nil {
		log.Printf("Error initializing iRacing client: %v", err)
		return
	}

	// Mark as initialized
	initialized = true
}

func CallApi(endpoint string, params map[string]interface{}) (*http.Response, error) {
	// If not initialized, return error
	if !initialized {
		return nil, fmt.Errorf("function not initialized")
	}

	// TODO: fix parameters type
	return iracingClient.Client.Get(endpoint, params)
}
