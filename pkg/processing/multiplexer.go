package processing

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub/v2"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/bus"
	"riccardotornesello.it/sharetelemetry/iracing/pkg/database"
)

func MultiplexProcessing(db *database.DB, ctx context.Context, pub *pubsub.Publisher, msgData *bus.ApiResponse) error {
	var err error

	switch msgData.Endpoint {
	case "/data/results/get":
		err = processSessionResults(db, msgData, ctx, pub)
		if err != nil {
			return fmt.Errorf("failed to process session results: %w", err)
		}

	case "/data/results/lap_data":
		err = processSessionLaps(db, msgData)
		if err != nil {
			return fmt.Errorf("failed to process session laps: %w", err)
		}

	case "/data/league/season_sessions":
		err = processLeagueSeasonSessions(db, msgData, ctx, pub)
		if err != nil {
			return fmt.Errorf("failed to process league season sessions: %w", err)
		}

	default:
		log.Printf("Skipping unknown endpoint: %s", msgData.Endpoint)
		return nil
	}

	return nil
}
