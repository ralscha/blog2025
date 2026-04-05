package main

import (
	"context"
	"fmt"
	"log"

	"timescaledbdemo/internal/store/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type step struct {
	name string
	run  func(*sqlc.Queries, context.Context) error
}

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:postgres@localhost:5432/timescaledb"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	if err = pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	queries := sqlc.New(pool)

	fmt.Println("=== TimescaleDB Cleanup Demo ===")

	steps := []step{
		{
			name: "Drop continuous aggregate view",
			run:  (*sqlc.Queries).DropSensorHourlyView,
		},
		{
			name: "Drop sensor_data hypertable",
			run:  (*sqlc.Queries).DropSensorDataTable,
		},
		{
			name: "Drop stock_prices hypertable",
			run:  (*sqlc.Queries).DropStockPricesTable,
		},
	}

	for _, s := range steps {
		if err = s.run(queries, ctx); err != nil {
			log.Fatalf("%s failed: %v\n", s.name, err)
		}
		fmt.Printf("OK: %s\n", s.name)
	}
}
