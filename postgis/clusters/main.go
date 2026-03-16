package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"starbucks/internal/demo"
)

func main() {
	var countryCode string
	var clusters int

	flag.StringVar(&countryCode, "country", "JP", "country code to cluster")
	flag.IntVar(&clusters, "k", 4, "number of K-means clusters")
	flag.Parse()

	pool, err := demo.Open(context.Background())
	check(err)
	defer pool.Close()

	results, err := demo.ClusterByCountry(context.Background(), pool, countryCode, int32(clusters))
	check(err)

	if len(results) == 0 {
		fmt.Printf("No stores found for country %s.\n", countryCode)
		return
	}

	for _, result := range results {
		fmt.Printf(
			"cluster=%d stores=%d center=%.5f, %.5f\n",
			result.ClusterID,
			result.StoreCount,
			result.CenterLatitude,
			result.CenterLongitude,
		)
	}
}

func check(e error) {
	if e != nil {
		log.Panicln(e)
	}
}
