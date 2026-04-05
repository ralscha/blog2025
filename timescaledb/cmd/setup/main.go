package main

import (
	"context"
	"fmt"
	"log"

	"timescaledbdemo/internal/store/sqlc"

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

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	fmt.Println("Successfully connected to TimescaleDB!")
	queries := sqlc.New(pool)

	if err = queries.EnableTimescaleExtension(ctx); err != nil {
		log.Fatalf("Error enabling timescaledb extension: %v\n", err)
	}

	if err = queries.CreateSensorDataTable(ctx); err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}

	if err = queries.CreateSensorDataIndex(ctx); err != nil {
		log.Fatalf("Error creating index: %v\n", err)
	}

	if err = queries.CreateSensorDataHypertable(ctx); err != nil {
		log.Fatalf("Error creating hypertable: %v\n", err)
	}

	fmt.Println("Hypertable 'sensor_data' created successfully!")

	if err = queries.SetSensorDataChunkInterval(ctx); err != nil {
		log.Fatalf("Error setting chunk interval: %v\n", err)
	}

	fmt.Println("Chunk interval set to 1 day.")

	if err = queries.InsertSampleSensorData(ctx); err != nil {
		log.Fatalf("Error inserting sample data: %v\n", err)
	}

	fmt.Println("Inserted sample telemetry rows for the last 24 hours.")

	if err = queries.CreateSensorHourlyCagg(ctx); err != nil {
		log.Fatalf("Error creating continuous aggregate: %v\n", err)
	}

	if err = queries.AddSensorHourlyCaggPolicy(ctx); err != nil {
		log.Fatalf("Error creating continuous aggregate policy: %v\n", err)
	}

	if err = queries.SetSensorDataColumnstore(ctx); err != nil {
		log.Fatalf("Error enabling columnstore: %v\n", err)
	}

	fmt.Println("Columnstore enabled with sensor_id segmentation and time DESC ordering.")

	if err = queries.AddSensorDataColumnstorePolicy(ctx); err != nil {
		log.Fatalf("Error creating columnstore policy: %v\n", err)
	}

	fmt.Println("Columnstore policy created for chunks older than 7 days.")

	if err = queries.AddSensorDataRetentionPolicy(ctx); err != nil {
		log.Fatalf("Error creating retention policy: %v\n", err)
	}

	if err = queries.AnalyzeSensorData(ctx); err != nil {
		log.Fatalf("Error analyzing hypertable: %v\n", err)
	}

	stats, err := queries.GetSensorStatistics(ctx)
	if err != nil {
		log.Fatalf("Error querying data: %v\n", err)
	}

	fmt.Println("\n=== Sensor Statistics ===")
	for _, row := range stats {
		fmt.Printf("Sensor %s: Count=%d, Avg Temp=%.2f°C, Avg Humidity=%.2f%%\n",
			row.SensorID, row.Count, row.AvgTemp, row.AvgHumidity)
	}
}
