package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/league/season_sessions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/firestore"
)

type League struct {
	Seasons map[string]LeagueSeason `firestore:"seasons"`
}

type LeagueSeason struct {
	SessionsParsed map[string]LeagueSeasonSession `firestore:"sessions_parsed"`
}

type LeagueSeasonSession struct {
	EntryCount int64     `firestore:"entry_count"` // TODO: fix always zero, should be filled from the API
	LaunchAt   time.Time `firestore:"launch_at"`
	TrackID    int64     `firestore:"track_id"`
}

func ProcessLeagueSeasonSessions(fc *firestore.FirestoreClient, msgData *bus.ApiResponse, ctx context.Context, pub *pubsub.Publisher) error {
	var err error

	body := []byte(msgData.Body)

	leagueID := msgData.Params["league_id"]
	seasonID := msgData.Params["season_id"]

	// Convert to the IRacing's API response
	var iracingSeasonSessions season_sessions.LeagueSeasonSessionsResponse
	err = json.Unmarshal(body, &iracingSeasonSessions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	// Get the league in Firestore
	league, err := firestore.Get[League](fc, "leagues", leagueID)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("failed to get league from Firestore: %w", err)
		} else {
			league = &League{}
		}
	}
	if league.Seasons == nil {
		league.Seasons = make(map[string]LeagueSeason)
	}
	if _, ok := league.Seasons[seasonID]; !ok {
		league.Seasons[seasonID] = LeagueSeason{}
	}

	// Find the missing subsessions
	alreadyParsedSubsessions := make(map[string]interface{})
	for parsedSubsessionID, _ := range league.Seasons[seasonID].SessionsParsed {
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
	newSessionsParsed := make(map[string]LeagueSeasonSession)
	for _, iracingSession := range iracingSeasonSessions.Sessions {
		newSessionsParsed[fmt.Sprintf("%d", iracingSession.SubsessionID)] = LeagueSeasonSession{
			EntryCount: iracingSession.EntryCount,
			LaunchAt:   iracingSession.LaunchAt.Time,
			TrackID:    iracingSession.Track.TrackID,
		}
	}

	season := league.Seasons[seasonID]
	season.SessionsParsed = newSessionsParsed
	league.Seasons[seasonID] = season

	err = firestore.Set(fc, "leagues", leagueID, league)
	if err != nil {
		return fmt.Errorf("failed to update league in Firestore: %w", err)
	}

	log.Printf("Published %d sessions requests for league ID: %s", len(pubsubRequests), leagueID)
	return nil
}
