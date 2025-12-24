package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/league/season_sessions"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/bus"
	"github.com/riccardotornesello/sharetelemetry-iracing-scraper/pkg/database"
)

type SeasonDoc struct {
	Meta   database.Meta `bson:"meta,omitempty"`
	Status SeasonStatus  `bson:"status,omitempty"`
}

type SeasonStatus struct {
	ParsedSessions map[string]SeasonStatusSession `bson:"parsed_sessions,omitempty"`
}

type SeasonStatusSession struct {
	LaunchAt *time.Time `bson:"launch_at,omitempty"`
	TrackID  *int64     `bson:"track_id,omitempty"`
}

func generateSeasonDocumentName(leagueID int64, seasonID int64) string {
	return fmt.Sprintf("league_%d_season_%d", leagueID, seasonID)
}

func getOrCreateSeasonDocument(db *database.DB, leagueID int64, seasonID int64) (*SeasonDoc, error) {
	var season SeasonDoc

	document_name := generateSeasonDocumentName(leagueID, seasonID)

	err := db.GetOne(SeasonCollection, SeasonKind, document_name, &season)
	if err != nil {
		if err == database.ErrNotFound {
			season = SeasonDoc{
				Meta: database.Meta{
					Version:   0,
					CreatedAt: time.Now().UTC(),

					Kind:   SeasonKind,
					Name:   document_name,
					Labels: map[string]interface{}{},
				},
				Status: SeasonStatus{
					ParsedSessions: make(map[string]SeasonStatusSession),
				},
			}

			err = db.Create(SeasonCollection, &season)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &season, nil
}

func saveSeasonDocument(db *database.DB, season *SeasonDoc) error {
	season.Meta.Version += 1
	return db.Update(SeasonCollection, SeasonKind, season.Meta.Name, season.Meta.Version-1, season)
}

func processLeagueSeasonSessions(db *database.DB, msgData *bus.ApiResponse, ctx context.Context, pub *pubsub.Publisher) error {
	var err error

	body := []byte(msgData.Body)

	leagueID, err := strconv.ParseInt(msgData.Params["league_id"], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid league_id parameter: %w", err)
	}

	seasonID, err := strconv.ParseInt(msgData.Params["season_id"], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid season_id parameter: %w", err)
	}

	// Convert to the IRacing's API response
	var iracingSeasonSessions season_sessions.LeagueSeasonSessionsResponse
	err = json.Unmarshal(body, &iracingSeasonSessions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	// Get the season from the database
	season, err := getOrCreateSeasonDocument(db, leagueID, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get or create season document: %w", err)
	}

	// Find the missing subsessions
	alreadyParsedSubsessions := make(map[string]interface{})
	for parsedSubsessionID, _ := range season.Status.ParsedSessions {
		alreadyParsedSubsessions[parsedSubsessionID] = nil
	}

	var missingSubsessionIds []string
	for _, iracingSession := range iracingSeasonSessions.Sessions {
		if _, ok := alreadyParsedSubsessions[fmt.Sprintf("%d", iracingSession.SubsessionID)]; !ok {
			missingSubsessionIds = append(missingSubsessionIds, fmt.Sprintf("%d", iracingSession.SubsessionID))
		}
	}

	// Send request to parse the sessions
	var pubsubRequests [][]byte
	var pubsubResults []*pubsub.PublishResult

	for _, subsessionID := range missingSubsessionIds {
		apiRequest := bus.ApiRequest{
			Endpoint: "/data/results/get",
			Params: map[string]string{
				"subsession_id":    subsessionID,
				"include_licenses": "false",
			},
		}

		data, err := json.Marshal(apiRequest)
		if err != nil {
			log.Printf("Failed to marshal sessions request: %v", err)
			continue
		}

		pubsubRequests = append(pubsubRequests, data)
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
			log.Printf("Failed to publish sessions request message: %v", err)
		}
	}

	// Update the league season with the newly parsed sessions
	// Convert the sessions to a map for easier storage
	season.Status.ParsedSessions = make(map[string]SeasonStatusSession)
	for _, iracingSession := range iracingSeasonSessions.Sessions {
		season.Status.ParsedSessions[fmt.Sprintf("%d", iracingSession.SubsessionID)] = SeasonStatusSession{
			LaunchAt: &iracingSession.LaunchAt.Time,
			TrackID:  &iracingSession.Track.TrackID,
		}
	}

	// Update the labels
	season.Meta.Labels["league_id"] = leagueID
	season.Meta.Labels["season_id"] = seasonID

	err = saveSeasonDocument(db, season)
	if err != nil {
		return fmt.Errorf("failed to update season document: %w", err)
	}

	log.Printf("Published %d sessions requests for league ID: %d", len(pubsubRequests), leagueID)
	return nil
}
