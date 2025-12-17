package processing

import (
	"encoding/json"
	"fmt"
	"log"

	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/firestore"
)

func ProcessSessionLaps(msgData *bus.ApiResponse) error {
	var err error

	body := []byte(msgData.Body)

	// Convert to a map for Firestore
	// NOTE: this is needed to keep the key names as in the original response
	var lapMapData map[string]interface{}
	err = json.Unmarshal(body, &lapMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body to map: %w", err)
	}

	// Save to firestore
	subsessionID := msgData.Params["subsession_id"]
	simsessionNumber := msgData.Params["simsession_number"]
	custID := msgData.Params["cust_id"]

	fieldKey := fmt.Sprintf("spec.laps.%s.%s", simsessionNumber, custID)

	err = firestore.Update("sessions", subsessionID, fieldKey, lapMapData)
	if err != nil {
		return fmt.Errorf("failed to update session results (%s/%s) in Firestore: %w", subsessionID, fieldKey, err)
	}

	log.Printf("Successfully saved laps for %s/%s", subsessionID, fieldKey)

	return nil
}
