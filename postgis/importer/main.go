package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"starbucks/internal/demo"
)

func main() {
	var csvPath string
	var truncate bool

	flag.StringVar(&csvPath, "csv", "starbucks.csv", "path to the Starbucks CSV file")
	flag.BoolVar(&truncate, "truncate", true, "truncate the stores table before importing")
	flag.Parse()

	ctx := context.Background()
	pool, err := demo.Open(ctx)
	check(err)
	defer pool.Close()

	check(demo.EnsureSchema(ctx, pool))
	if truncate {
		check(demo.TruncateStores(ctx, pool))
	}

	count, err := demo.ImportCSV(ctx, pool, csvPath)
	check(err)

	fmt.Printf("Imported %d stores into PostGIS.\n", count)
}

func check(e error) {
	if e != nil {
		log.Panicln(e)
	}
}
