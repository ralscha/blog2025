package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"timescaledbdemo/internal/store/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:postgres@localhost:5432/timescaledb"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	if err = queries.EnableTimescaleExtension(ctx); err != nil {
		log.Fatalf("Error enabling timescaledb extension: %v\n", err)
	}

	fmt.Println("=== Batch Insert Demo ===")

	if err = queries.CreateStockPricesTable(ctx); err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}

	if err = queries.CreateStockPricesIndex(ctx); err != nil {
		log.Fatalf("Error creating index: %v\n", err)
	}

	if err = queries.CreateStockPricesHypertable(ctx); err != nil {
		log.Fatalf("Error creating hypertable: %v\n", err)
	}

	symbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"}
	startTime := time.Now().Add(-24 * time.Hour)
	batchSize := 1000

	fmt.Printf("Inserting %d records...\n", batchSize)
	start := time.Now()

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("Error starting transaction: %v\n", err)
	}
	txQueries := queries.WithTx(tx)

	for i := range batchSize {
		timestamp := startTime.Add(time.Duration(i) * time.Minute)
		symbol := symbols[rand.Intn(len(symbols))]
		price := 100.0 + rand.Float64()*50.0
		volume := int32(rand.Intn(1000000) + 10000)

		err = txQueries.InsertStockPrice(ctx, sqlc.InsertStockPriceParams{
			Time: pgtype.Timestamptz{
				Time:  timestamp,
				Valid: true,
			},
			Symbol: symbol,
			Price:  &price,
			Volume: &volume,
		})
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("Error inserting data: %v\n", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Fatalf("Error committing transaction: %v\n", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Inserted %d records in %v (%.0f records/sec)\n\n",
		batchSize, elapsed, float64(batchSize)/elapsed.Seconds())

	fmt.Println("Stock price summary:")
	summaryRows, err := queries.GetStockPriceSummaryBySymbol(ctx)
	if err != nil {
		log.Fatalf("Error querying data: %v\n", err)
	}
	for _, row := range summaryRows {
		fmt.Printf("%s: Count=%d, Avg=$%.2f, Min=$%.2f, Max=$%.2f, Volume=%d\n",
			row.Symbol, row.Count, row.AvgPrice, row.MinPrice, row.MaxPrice, row.TotalVolume)
	}

	fmt.Println("\nPrice trends (hourly averages):")
	trends, err := queries.GetStockPriceTrendsLast6Hours(ctx)
	if err != nil {
		log.Fatalf("Error querying trends: %v\n", err)
	}

	var lastSymbol string
	for _, row := range trends {
		symbol := row.Symbol
		if symbol != lastSymbol {
			if lastSymbol != "" {
				fmt.Println()
			}
			fmt.Printf("%s:\n", symbol)
			lastSymbol = symbol
		}
		fmt.Printf("  %s: $%.2f\n", row.Hour.Time.Format("15:04"), row.AvgPrice)
	}
}
