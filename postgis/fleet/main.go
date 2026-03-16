package main

import (
	"context"
	"flag"
	"log"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"starbucks/internal/demo"
)

type truckRoute struct {
	TruckID    string
	DriverName string
	StartLat   float64
	StartLon   float64
	EndLat     float64
	EndLon     float64
	Phase      int
}

const routeCycleSteps = 24

func main() {
	var interval time.Duration

	flag.DurationVar(&interval, "interval", 3*time.Second, "time between truck position updates")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	pool, err := demo.Open(ctx)
	check(err)
	defer pool.Close()

	check(demo.EnsureSchema(ctx, pool))
	check(demo.EnsureGeofencingSchema(ctx, pool))
	check(demo.SeedHomeDepotGeofence(ctx, pool))

	routes := demoFleet()
	log.Printf("publishing %d truck updates every %s", len(routes), interval)

	step := 0
	check(publishFleet(ctx, pool, routes, step))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("fleet simulator stopped")
			return
		case <-ticker.C:
			step++
			check(publishFleet(ctx, pool, routes, step))
		}
	}
}

func publishFleet(ctx context.Context, pool *pgxpool.Pool, routes []truckRoute, step int) error {
	now := time.Now().UTC()
	for _, route := range routes {
		latitude, longitude := route.position(step)
		update := demo.TruckPositionUpdate{
			TruckID:    route.TruckID,
			DriverName: route.DriverName,
			Latitude:   latitude,
			Longitude:  longitude,
			UpdatedAt:  now,
		}

		if err := demo.UpsertTruckPosition(ctx, pool, update); err != nil {
			return err
		}
	}

	log.Printf("tick %02d published %d truck positions", step, len(routes))
	return nil
}

func (route truckRoute) position(step int) (float64, float64) {
	progressStep := (step + route.Phase) % (routeCycleSteps * 2)
	progress := float64(progressStep) / float64(routeCycleSteps)
	if progress > 1 {
		progress = 2 - progress
	}

	progress = math.Max(0, math.Min(1, progress))
	latitude := route.StartLat + ((route.EndLat - route.StartLat) * progress)
	longitude := route.StartLon + ((route.EndLon - route.StartLon) * progress)
	return latitude, longitude
}

func demoFleet() []truckRoute {
	return []truckRoute{
		{TruckID: "truck-01", DriverName: "Ava", StartLat: 47.5802, StartLon: -122.3415, EndLat: 47.5802, EndLon: -122.3270, Phase: 0},
		{TruckID: "truck-02", DriverName: "Ben", StartLat: 47.5768, StartLon: -122.3337, EndLat: 47.5843, EndLon: -122.3337, Phase: 3},
		{TruckID: "truck-03", DriverName: "Cara", StartLat: 47.5775, StartLon: -122.3408, EndLat: 47.5830, EndLon: -122.3283, Phase: 6},
		{TruckID: "truck-04", DriverName: "Diego", StartLat: 47.5835, StartLon: -122.3388, EndLat: 47.5770, EndLon: -122.3298, Phase: 9},
		{TruckID: "truck-05", DriverName: "Elena", StartLat: 47.5750, StartLon: -122.3460, EndLat: 47.5750, EndLon: -122.3390, Phase: 1},
		{TruckID: "truck-06", DriverName: "Finn", StartLat: 47.5849, StartLon: -122.3462, EndLat: 47.5849, EndLon: -122.3392, Phase: 5},
		{TruckID: "truck-07", DriverName: "Gia", StartLat: 47.5758, StartLon: -122.3268, EndLat: 47.5844, EndLon: -122.3268, Phase: 7},
		{TruckID: "truck-08", DriverName: "Hugo", StartLat: 47.5772, StartLon: -122.3450, EndLat: 47.5838, EndLon: -122.3420, Phase: 11},
		{TruckID: "truck-09", DriverName: "Iris", StartLat: 47.5794, StartLon: -122.3288, EndLat: 47.5860, EndLon: -122.3258, Phase: 13},
		{TruckID: "truck-10", DriverName: "Jules", StartLat: 47.5739, StartLon: -122.3378, EndLat: 47.5760, EndLon: -122.3298, Phase: 15},
	}
}

func check(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
