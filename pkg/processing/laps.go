package processing

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/riccardotornesello/irapi-go/pkg/api/results/lap_data"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/database"
)

type LapsDoc struct {
	Meta database.Meta `bson:"meta,omitempty"`
	Spec LapsSpec      `bson:"spec,omitempty"`
}

type LapsSpec struct {
	Data   map[string]interface{}   `bson:"data,omitempty"`
	Chunks []map[string]interface{} `bson:"chunks,omitempty"`
}

func generateLapsDocumentName(subsessionID int64, simsessionNumber int64, custID int64) string {
	return fmt.Sprintf("laps_%d_%d_%d", subsessionID, simsessionNumber, custID)
}

func getOrCreateLapsDocument(db *database.DB, subsessionID int64, simsessionNumber int64, custID int64) (*LapsDoc, error) {
	var laps LapsDoc

	document_name := generateLapsDocumentName(subsessionID, simsessionNumber, custID)

	err := db.GetOne(SessionCollection, LapsKind, document_name, &laps)
	if err != nil {
		if err == database.ErrNotFound {
			laps = LapsDoc{
				Meta: database.Meta{
					Version:   0,
					CreatedAt: time.Now().UTC(),

					Kind:   LapsKind,
					Name:   document_name,
					Labels: map[string]interface{}{},
				},
				Spec: LapsSpec{},
			}

			err = db.Create(SessionCollection, &laps)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &laps, nil
}

func saveLapsDocument(db *database.DB, laps *LapsDoc) error {
	laps.Meta.Version += 1
	return db.Update(SessionCollection, LapsKind, laps.Meta.Name, laps.Meta.Version-1, laps)
}

func processSessionLaps(db *database.DB, msgData *bus.ApiResponse) error {
	var err error

	body := []byte(msgData.Body)
	chunks := []byte(*msgData.Chunks)

	// Convert to the IRacing's API response
	var iRacingLaps lap_data.ResultsLapDataResponse
	err = json.Unmarshal(body, &iRacingLaps)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	subsessionID := iRacingLaps.SessionInfo.SubsessionID
	simsessionNumber := iRacingLaps.SessionInfo.SimsessionNumber
	custID := iRacingLaps.CustID
	trackID := iRacingLaps.SessionInfo.Track.TrackID
	carID := iRacingLaps.CarID

	// Get the laps document from the database
	lapsDoc, err := getOrCreateLapsDocument(db, subsessionID, simsessionNumber, custID)
	if err != nil {
		return fmt.Errorf("failed to get or create laps document: %w", err)
	}

	// Update the data and labels
	lapsDoc.Meta.Labels["subsession_id"] = subsessionID
	lapsDoc.Meta.Labels["simsession_number"] = simsessionNumber
	lapsDoc.Meta.Labels["cust_id"] = custID
	lapsDoc.Meta.Labels["track_id"] = trackID
	lapsDoc.Meta.Labels["car_id"] = carID

	var lapMapData map[string]interface{}
	err = json.Unmarshal(body, &lapMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body to map: %w", err)
	}

	var chunksMapData []map[string]interface{}
	err = json.Unmarshal(chunks, &chunksMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body to chunks map: %w", err)
	}

	lapsDoc.Spec.Data = lapMapData
	lapsDoc.Spec.Chunks = chunksMapData

	// Save to the database
	err = saveLapsDocument(db, lapsDoc)
	if err != nil {
		return fmt.Errorf("failed to save laps document: %w", err)
	}

	log.Printf("Successfully saved laps for %d/%d/%d", subsessionID, simsessionNumber, custID)

	return nil
}
