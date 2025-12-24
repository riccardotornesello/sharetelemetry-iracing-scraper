package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/results/get"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/bus"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/database"
)

type SessionDoc struct {
	Meta database.Meta `bson:"meta,omitempty"`
	Spec SessionSpec   `bson:"spec,omitempty"`
}

type SessionSpec struct {
	Data map[string]interface{} `bson:"data,omitempty"`
}

func generateSessionDocumentName(subsessionID int64) string {
	return fmt.Sprintf("session_%d", subsessionID)
}

func getOrCreateSessionDocument(db *database.DB, subsessionID int64) (*SessionDoc, error) {
	var session SessionDoc

	document_name := generateSessionDocumentName(subsessionID)

	err := db.GetOne(SessionCollection, SessionKind, document_name, &session)
	if err != nil {
		if err == database.ErrNotFound {
			session = SessionDoc{
				Meta: database.Meta{
					Version:   0,
					CreatedAt: time.Now().UTC(),

					Kind:   SessionKind,
					Name:   document_name,
					Labels: map[string]interface{}{},
				},
				Spec: SessionSpec{},
			}

			err = db.Create(SessionCollection, &session)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &session, nil
}

func saveSessionDocument(db *database.DB, session *SessionDoc) error {
	session.Meta.Version += 1
	return db.Update(SessionCollection, SessionKind, session.Meta.Name, session.Meta.Version-1, session)
}

func processSessionResults(db *database.DB, msgData *bus.ApiResponse, ctx context.Context, pub *pubsub.Publisher) error {
	var err error

	body := []byte(msgData.Body)

	// Convert to the IRacing's API response
	var iRacingSession get.ResultsGetResponse
	err = json.Unmarshal(body, &iRacingSession)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	subsessionID := iRacingSession.SubsessionID
	log.Printf("Processing results for subsession ID: %d", subsessionID)

	// Get the session from the database
	session, err := getOrCreateSessionDocument(db, subsessionID)
	if err != nil {
		return fmt.Errorf("failed to get or create session document: %w", err)
	}

	// Update the session document data and labels
	session.Meta.Labels["league_id"] = iRacingSession.LeagueID
	session.Meta.Labels["season_id"] = iRacingSession.SeasonID
	session.Meta.Labels["subsession_id"] = iRacingSession.SubsessionID
	session.Meta.Labels["track_id"] = iRacingSession.Track.TrackID

	err = json.Unmarshal(body, &session.Spec.Data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body to map: %w", err)
	}

	// Save to the database
	err = saveSessionDocument(db, session)
	if err != nil {
		return fmt.Errorf("failed to save session document: %w", err)
	}

	log.Printf("Successfully saved results for subsession ID: %d", subsessionID)

	// Send request to parse lap data
	var pubsubRequests [][]byte
	var pubsubResults []*pubsub.PublishResult

	for _, simsession := range iRacingSession.SessionResults {
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
