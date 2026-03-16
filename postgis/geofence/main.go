package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"starbucks/internal/demo"
)

func main() {
	var geofenceID string
	var interval time.Duration

	flag.StringVar(&geofenceID, "geofence", demo.HomeDepotGeofenceID, "geofence id to watch")
	flag.DurationVar(&interval, "interval", 2*time.Second, "poll interval for new geofence events")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	pool, err := demo.Open(ctx)
	check(err)
	defer pool.Close()

	check(demo.EnsureSchema(ctx, pool))
	check(demo.EnsureGeofencingSchema(ctx, pool))
	check(demo.SeedHomeDepotGeofence(ctx, pool))

	lastSeenAt := time.Now().UTC()
	var lastSeenID int64
	log.Printf("polling geofence events for %q every %s starting at %s", geofenceID, interval, lastSeenAt.Format(time.RFC3339))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("geofence poller stopped")
			return
		case <-ticker.C:
			lastSeenAt, lastSeenID = pollAndLogEvents(ctx, pool, geofenceID, lastSeenAt, lastSeenID)
		}
	}
}

func pollAndLogEvents(ctx context.Context, pool *pgxpool.Pool, geofenceID string, lastSeenAt time.Time, lastSeenID int64) (time.Time, int64) {
	const batchSize = int32(100)

	for {
		events, err := demo.ListTruckGeofenceEventsSince(ctx, pool, geofenceID, lastSeenAt, lastSeenID, batchSize)
		if err != nil {
			log.Printf("failed to poll geofence events: %v", err)
			return lastSeenAt, lastSeenID
		}

		if len(events) == 0 {
			return lastSeenAt, lastSeenID
		}

		for _, event := range events {
			switch event.EventType {
			case "entered":
				log.Printf("ALERT: driver %s (%s) entered %s at %.5f, %.5f", event.DriverName, event.TruckID, event.GeofenceName, event.Latitude, event.Longitude)
			case "exited":
				log.Printf("INFO: driver %s (%s) exited %s", event.DriverName, event.TruckID, event.GeofenceName)
			default:
				log.Printf("EVENT: driver %s (%s) %s %s", event.DriverName, event.TruckID, event.EventType, event.GeofenceName)
			}

			lastSeenAt = event.OccurredAt
			lastSeenID = event.ID
		}

		if len(events) < int(batchSize) {
			return lastSeenAt, lastSeenID
		}
	}
}

func check(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
