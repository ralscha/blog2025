package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"starbucks/internal/demo"
)

func main() {
	var fromLatitude float64
	var fromLongitude float64
	var toLatitude float64
	var toLongitude float64
	var distanceMeters float64
	var limit int

	flag.Float64Var(&fromLatitude, "from-lat", 35.6585, "origin latitude")
	flag.Float64Var(&fromLongitude, "from-lon", 139.7013, "origin longitude")
	flag.Float64Var(&toLatitude, "to-lat", 35.6895, "destination latitude")
	flag.Float64Var(&toLongitude, "to-lon", 139.6917, "destination longitude")
	flag.Float64Var(&distanceMeters, "distance", 600, "maximum distance from the route in meters")
	flag.IntVar(&limit, "limit", 10, "maximum number of stores to return")
	flag.Parse()

	pool, err := demo.Open(context.Background())
	check(err)
	defer pool.Close()

	stores, err := demo.StoresAlongCorridor(
		context.Background(),
		pool,
		fromLatitude,
		fromLongitude,
		toLatitude,
		toLongitude,
		distanceMeters,
		int32(limit),
	)
	check(err)

	if len(stores) == 0 {
		fmt.Println("nothing found")
		return
	}

	for _, store := range stores {
		fmt.Printf(
			"%s | %s | %s | %.5f, %.5f | route-distance=%dm\n",
			store.StoreNumber,
			store.CountryCode,
			firstNonEmpty(store.City, store.StreetAddress),
			store.Latitude,
			store.Longitude,
			store.DistanceToRouteMeters,
		)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func check(e error) {
	if e != nil {
		log.Panicln(e)
	}
}
