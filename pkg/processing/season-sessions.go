package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub/v2"
	"github.com/riccardotornesello/irapi-go/pkg/api/league/season_sessions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/firestore"
)

type League struct {
	Seasons map[string]LeagueSeason
}

type LeagueSeason struct {
	SessionsParsed []int64
}

func ProcessLeagueSeasonSessions(msgData *bus.ApiResponse, ctx context.Context, pub *pubsub.Publisher) error {
	var err error

	body := []byte(msgData.Body)

	leagueID := msgData.Params["league_id"]
	seasonID := msgData.Params["season_id"]

	// Convert to the IRacing's API response
	var seasonSessions season_sessions.LeagueSeasonSessionsResponse
	err = json.Unmarshal(body, &seasonSessions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response body: %w", err)
	}

	// Get the league
	league, err := firestore.Get[League]("leagues", leagueID)
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
	subsessionIds := make([]int64, len(seasonSessions.Sessions))
	for i, session := range seasonSessions.Sessions {
		subsessionIds[i] = session.SubsessionID
	}

	parsedSubsessions := make(map[int64]bool)
	for _, parsedSubsessionID := range league.Seasons[seasonID].SessionsParsed {
		parsedSubsessions[parsedSubsessionID] = true
	}

	var missingSubsessionIds []int64
	for _, subsessionID := range subsessionIds {
		if _, ok := parsedSubsessions[subsessionID]; !ok {
			missingSubsessionIds = append(missingSubsessionIds, subsessionID)
		}
	}

	// Send request to parse the sessions
	var pubsubRequests [][]byte
	var pubsubResults []*pubsub.PublishResult

	for _, subsessionID := range missingSubsessionIds {
		apiRequest := bus.ApiRequest{
			Endpoint: "/data/results/get",
			Params: map[string]string{
				"subsession_id":    fmt.Sprintf("%d", subsessionID),
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
	season := league.Seasons[seasonID]
	season.SessionsParsed = append(season.SessionsParsed, missingSubsessionIds...)
	league.Seasons[seasonID] = season

	err = firestore.Set("leagues", leagueID, league)
	if err != nil {
		return fmt.Errorf("failed to update league in Firestore: %w", err)
	}

	log.Printf("Published %d sessions requests for league ID: %s", len(pubsubRequests), leagueID)
	return nil
}
