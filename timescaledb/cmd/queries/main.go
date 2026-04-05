package main

import (
	"context"
	"fmt"
	"log"

	"timescaledbdemo/internal/store/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const gapfillDemoQuery = `
SELECT time_bucket_gapfill(
					 '10 minutes',
					 time,
					 start => NOW() - INTERVAL '30 minutes',
					 finish => NOW()
			 )::timestamptz AS bucket,
			 sensor_id,
			 CASE
					 WHEN COUNT(temperature) = 0 THEN NULL::float8
					 ELSE AVG(temperature)::float8
			 END AS avg_temp
FROM sensor_data
WHERE sensor_id = $1
	AND time >= NOW() - INTERVAL '30 minutes'
	AND time < NOW()
GROUP BY bucket, sensor_id
ORDER BY bucket DESC
LIMIT 5
`

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:postgres@localhost:5432/timescaledb"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	fmt.Println("=== TimescaleDB Time-Series Queries Demo ===")

	fmt.Println("1. Average temperature per hour (last 24 hours):")
	hourly, err := queries.GetHourlyAverageTemperatureLast24h(ctx)
	if err != nil {
		log.Fatalf("Error in time-bucket query: %v\n", err)
	}
	for _, row := range hourly {
		fmt.Printf("  %s: %.2f°C\n", row.Hour.Time.Format("2006-01-02 15:04"), row.AvgTemp)
	}

	fmt.Println("\n2. Latest readings per sensor:")
	latest, err := queries.GetLatestReadingsPerSensor(ctx)
	if err != nil {
		log.Fatalf("Error querying latest readings: %v\n", err)
	}
	for _, row := range latest {
		temp := 0.0
		humidity := 0.0
		if row.Temperature != nil {
			temp = *row.Temperature
		}
		if row.Humidity != nil {
			humidity = *row.Humidity
		}
		fmt.Printf("  Sensor %s: %.2f°C, %.2f%% (at %s)\n",
			row.SensorID, temp, humidity, row.Time.Time.Format("15:04:05"))
	}

	fmt.Println("\n3. 5-minute averages with min/max:")
	fiveMinute, err := queries.GetFiveMinuteTemperatureStatsLastHour(ctx)
	if err != nil {
		log.Fatalf("Error in downsampling query: %v\n", err)
	}
	for _, row := range fiveMinute {
		fmt.Printf("  %s: Avg=%.2f°C, Min=%.2f°C, Max=%.2f°C\n",
			row.Bucket.Time.Format("15:04"), row.AvgTemp, row.MinTemp, row.MaxTemp)
	}

	fmt.Println("\n4. Gap-filled data (every 10 minutes):")
	rows, err := pool.Query(ctx, gapfillDemoQuery, "sensor_1")
	if err != nil {
		log.Fatalf("Error in gap-fill query: %v\n", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bucket pgtype.Timestamptz
		var sensorID string
		var avgTemp *float64
		if err := rows.Scan(&bucket, &sensorID, &avgTemp); err != nil {
			log.Fatalf("Error scanning gap-fill row: %v\n", err)
		}
		if avgTemp == nil {
			fmt.Printf("  %s: NULL\n", bucket.Time.Format("15:04"))
			continue
		}
		fmt.Printf("  %s: %.2f°C\n", bucket.Time.Format("15:04"), *avgTemp)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating gap-fill rows: %v\n", err)
	}

	fmt.Println("\n5. Query continuous aggregate (last 7 days):")
	agg, err := queries.GetSensorHourlyAggregateLast7Days(ctx)
	if err != nil {
		log.Fatalf("Error querying continuous aggregate: %v\n", err)
	}
	for _, row := range agg {
		fmt.Printf("  %s | %s: Avg=%.2f°C Min=%.2f°C Max=%.2f°C Samples=%d\n",
			row.Bucket.Time.Format("2006-01-02 15:04"), row.SensorID, row.AvgTemp, row.MinTemp, row.MaxTemp, row.SampleCount)
	}
}
