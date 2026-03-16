package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"starbucks/internal/demo"
)

func main() {
	var latitude float64
	var longitude float64
	var radiusMeters float64
	var limit int

	flag.Float64Var(&latitude, "lat", 35.6895, "latitude of the search point")
	flag.Float64Var(&longitude, "lon", 139.6917, "longitude of the search point")
	flag.Float64Var(&radiusMeters, "radius", 1500, "search radius in meters")
	flag.IntVar(&limit, "limit", 5, "maximum number of stores to return")
	flag.Parse()

	pool, err := demo.Open(context.Background())
	check(err)
	defer pool.Close()

	stores, err := demo.Nearby(context.Background(), pool, latitude, longitude, radiusMeters, int32(limit))
	check(err)

	if len(stores) == 0 {
		fmt.Println("nothing found")
		return
	}

	for _, store := range stores {
		fmt.Printf(
			"%s | %s | %s | %.5f, %.5f | %dm\n",
			store.StoreNumber,
			store.CountryCode,
			firstNonEmpty(store.City, store.StreetAddress),
			store.Latitude,
			store.Longitude,
			store.DistanceMeters,
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
