package demo

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/starbucks?sslmode=disable"

const HomeDepotGeofenceID = "home-depot-sodo"

type NearbyStore struct {
	StoreNumber    string  `json:"storeNumber"`
	CountryCode    string  `json:"countryCode"`
	City           string  `json:"city"`
	StreetAddress  string  `json:"streetAddress"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	DistanceMeters int64   `json:"distanceMeters"`
}

type ClusterResult struct {
	ClusterID       int32   `json:"clusterID"`
	StoreCount      int32   `json:"storeCount"`
	CenterLatitude  float64 `json:"centerLatitude"`
	CenterLongitude float64 `json:"centerLongitude"`
}

type CorridorStore struct {
	StoreNumber           string  `json:"storeNumber"`
	CountryCode           string  `json:"countryCode"`
	City                  string  `json:"city"`
	StreetAddress         string  `json:"streetAddress"`
	Latitude              float64 `json:"latitude"`
	Longitude             float64 `json:"longitude"`
	DistanceToRouteMeters int64   `json:"distanceToRouteMeters"`
}

type CountrySummary struct {
	CountryCode string `json:"countryCode"`
	StoreCount  int32  `json:"storeCount"`
}

type TruckPositionUpdate struct {
	TruckID    string    `json:"truck_id"`
	DriverName string    `json:"driver_name"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TruckGeofenceStatus struct {
	TruckID      string    `json:"truckID"`
	DriverName   string    `json:"driverName"`
	GeofenceID   string    `json:"geofenceID"`
	GeofenceName string    `json:"geofenceName"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Inside       bool      `json:"inside"`
}

type GeofenceArea struct {
	GeofenceID string          `json:"geofenceID"`
	Name       string          `json:"name"`
	Category   string          `json:"category"`
	Geometry   json.RawMessage `json:"geometry"`
}

type TruckGeofenceEvent struct {
	ID           int64     `json:"id"`
	TruckID      string    `json:"truckID"`
	DriverName   string    `json:"driverName"`
	GeofenceID   string    `json:"geofenceID"`
	GeofenceName string    `json:"geofenceName"`
	EventType    string    `json:"eventType"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	OccurredAt   time.Time `json:"occurredAt"`
}

func DatabaseURL() string {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return defaultDatabaseURL
	}

	return databaseURL
}

func Open(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS stores (
	store_number TEXT PRIMARY KEY,
	country_code TEXT NOT NULL,
	ownership_type_code TEXT,
	schedule TEXT,
	slug TEXT,
	latitude DOUBLE PRECISION NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	street_address_line1 TEXT,
	street_address_line2 TEXT,
	street_address_line3 TEXT,
	city TEXT,
	country_subdivision_code TEXT,
	postal_code TEXT,
	current_time_offset INTEGER,
	windows_time_zone_id TEXT,
	olson_time_zone_id TEXT,
	location geometry(Point, 4326) GENERATED ALWAYS AS (
		ST_Point(longitude, latitude, 4326)
	) STORED,
	geog geography(Point, 4326) GENERATED ALWAYS AS (
		ST_Point(longitude, latitude, 4326)::geography
	) STORED
);

CREATE INDEX IF NOT EXISTS stores_geog_idx ON stores USING GIST (geog);
CREATE INDEX IF NOT EXISTS stores_country_code_idx ON stores (country_code);
CREATE INDEX IF NOT EXISTS stores_city_idx ON stores (city);
`

	_, err := pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	return nil
}

func EnsureGeofencingSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS geofences (
	geofence_id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	category TEXT NOT NULL,
	area geometry(Polygon, 4326) NOT NULL
);

CREATE INDEX IF NOT EXISTS geofences_area_idx ON geofences USING GIST (area);

CREATE TABLE IF NOT EXISTS truck_latest_positions (
	truck_id TEXT PRIMARY KEY,
	driver_name TEXT NOT NULL,
	latitude DOUBLE PRECISION NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	location geometry(Point, 4326) GENERATED ALWAYS AS (
		ST_Point(longitude, latitude, 4326)
	) STORED
);

CREATE INDEX IF NOT EXISTS truck_latest_positions_location_idx ON truck_latest_positions USING GIST (location);

CREATE TABLE IF NOT EXISTS truck_geofence_events (
	id BIGSERIAL PRIMARY KEY,
	truck_id TEXT NOT NULL,
	driver_name TEXT NOT NULL,
	geofence_id TEXT NOT NULL REFERENCES geofences (geofence_id),
	geofence_name TEXT NOT NULL,
	event_type TEXT NOT NULL CHECK (event_type IN ('entered', 'exited')),
	latitude DOUBLE PRECISION NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	occurred_at TIMESTAMPTZ NOT NULL,
	location geometry(Point, 4326) GENERATED ALWAYS AS (
		ST_Point(longitude, latitude, 4326)
	) STORED
);

CREATE INDEX IF NOT EXISTS truck_geofence_events_geofence_time_idx ON truck_geofence_events (geofence_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS truck_geofence_events_truck_time_idx ON truck_geofence_events (truck_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS truck_geofence_events_location_idx ON truck_geofence_events USING GIST (location);

CREATE OR REPLACE FUNCTION sync_truck_geofence_events() RETURNS trigger AS $$
DECLARE
	geofence_row RECORD;
	last_event_type TEXT;
	currently_inside BOOLEAN;
BEGIN
	FOR geofence_row IN
		SELECT geofence_id, name, ST_Contains(area, NEW.location) AS is_inside
		FROM geofences
	LOOP
		currently_inside := geofence_row.is_inside;

		SELECT event_type
		INTO last_event_type
		FROM truck_geofence_events
		WHERE truck_id = NEW.truck_id
			AND geofence_id = geofence_row.geofence_id
		ORDER BY occurred_at DESC, id DESC
		LIMIT 1;

		IF (last_event_type IS NULL AND currently_inside)
			OR (last_event_type = 'exited' AND currently_inside)
			OR (last_event_type = 'entered' AND NOT currently_inside)
		THEN
			INSERT INTO truck_geofence_events (
				truck_id,
				driver_name,
				geofence_id,
				geofence_name,
				event_type,
				latitude,
				longitude,
				occurred_at
			)
			VALUES (
				NEW.truck_id,
				NEW.driver_name,
				geofence_row.geofence_id,
				geofence_row.name,
				CASE WHEN currently_inside THEN 'entered' ELSE 'exited' END,
				NEW.latitude,
				NEW.longitude,
				NEW.updated_at
			);
		END IF;
	END LOOP;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS truck_latest_positions_sync_geofences ON truck_latest_positions;
CREATE TRIGGER truck_latest_positions_sync_geofences
AFTER INSERT OR UPDATE ON truck_latest_positions
FOR EACH ROW
EXECUTE FUNCTION sync_truck_geofence_events();
`

	_, err := pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("ensure geofencing schema: %w", err)
	}

	return nil
}

func SeedHomeDepotGeofence(ctx context.Context, pool *pgxpool.Pool) error {
	const polygonWKT = "POLYGON((-122.3375 47.5785, -122.3375 47.5825, -122.3295 47.5825, -122.3295 47.5785, -122.3375 47.5785))"

	_, err := pool.Exec(ctx, `
INSERT INTO geofences (geofence_id, name, category, area)
VALUES ($1, $2, $3, ST_GeomFromText($4, 4326))
ON CONFLICT (geofence_id) DO UPDATE
SET name = EXCLUDED.name,
	category = EXCLUDED.category,
	area = EXCLUDED.area
`, HomeDepotGeofenceID, "Home Depot", "retail-yard", polygonWKT)
	if err != nil {
		return fmt.Errorf("seed Home Depot geofence: %w", err)
	}

	return nil
}

func UpsertTruckPosition(ctx context.Context, pool *pgxpool.Pool, update TruckPositionUpdate) error {
	_, err := pool.Exec(ctx, `
INSERT INTO truck_latest_positions (truck_id, driver_name, latitude, longitude, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (truck_id) DO UPDATE
SET driver_name = EXCLUDED.driver_name,
	latitude = EXCLUDED.latitude,
	longitude = EXCLUDED.longitude,
	updated_at = EXCLUDED.updated_at
`, update.TruckID, update.DriverName, update.Latitude, update.Longitude, update.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert truck position: %w", err)
	}

	return nil
}

func GetTruckGeofenceStatus(ctx context.Context, pool *pgxpool.Pool, truckID string, geofenceID string) (TruckGeofenceStatus, error) {
	const query = `
SELECT
	t.truck_id,
	t.driver_name,
	g.geofence_id,
	g.name,
	t.latitude,
	t.longitude,
	t.updated_at,
	ST_Contains(g.area, t.location) AS inside
FROM truck_latest_positions t
JOIN geofences g ON g.geofence_id = $2
WHERE t.truck_id = $1
`

	var status TruckGeofenceStatus
	err := pool.QueryRow(ctx, query, truckID, geofenceID).Scan(
		&status.TruckID,
		&status.DriverName,
		&status.GeofenceID,
		&status.GeofenceName,
		&status.Latitude,
		&status.Longitude,
		&status.UpdatedAt,
		&status.Inside,
	)
	if err != nil {
		return TruckGeofenceStatus{}, fmt.Errorf("get truck geofence status: %w", err)
	}

	return status, nil
}

func ListTruckGeofenceStatuses(ctx context.Context, pool *pgxpool.Pool, geofenceID string) ([]TruckGeofenceStatus, error) {
	const query = `
SELECT
	t.truck_id,
	t.driver_name,
	g.geofence_id,
	g.name,
	t.latitude,
	t.longitude,
	t.updated_at,
	ST_Contains(g.area, t.location) AS inside
FROM truck_latest_positions t
JOIN geofences g ON g.geofence_id = $1
ORDER BY t.truck_id
`

	rows, err := pool.Query(ctx, query, geofenceID)
	if err != nil {
		return nil, fmt.Errorf("list truck geofence statuses: %w", err)
	}
	defer rows.Close()

	statuses := make([]TruckGeofenceStatus, 0)
	for rows.Next() {
		var status TruckGeofenceStatus
		if err := rows.Scan(
			&status.TruckID,
			&status.DriverName,
			&status.GeofenceID,
			&status.GeofenceName,
			&status.Latitude,
			&status.Longitude,
			&status.UpdatedAt,
			&status.Inside,
		); err != nil {
			return nil, fmt.Errorf("scan truck geofence status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate truck geofence statuses: %w", err)
	}

	return statuses, nil
}

func ListGeofences(ctx context.Context, pool *pgxpool.Pool) ([]GeofenceArea, error) {
	const query = `
SELECT
	geofence_id,
	name,
	category,
	ST_AsGeoJSON(area)::TEXT AS geometry
FROM geofences
ORDER BY geofence_id
`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list geofences: %w", err)
	}
	defer rows.Close()

	areas := make([]GeofenceArea, 0)
	for rows.Next() {
		var area GeofenceArea
		var geometryText string
		if err := rows.Scan(&area.GeofenceID, &area.Name, &area.Category, &geometryText); err != nil {
			return nil, fmt.Errorf("scan geofence: %w", err)
		}
		area.Geometry = json.RawMessage(geometryText)
		areas = append(areas, area)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofences: %w", err)
	}

	return areas, nil
}

func GetGeofence(ctx context.Context, pool *pgxpool.Pool, geofenceID string) (GeofenceArea, error) {
	const query = `
SELECT
	geofence_id,
	name,
	category,
	ST_AsGeoJSON(area)::TEXT AS geometry
FROM geofences
WHERE geofence_id = $1
`

	var area GeofenceArea
	var geometryText string
	err := pool.QueryRow(ctx, query, geofenceID).Scan(
		&area.GeofenceID,
		&area.Name,
		&area.Category,
		&geometryText,
	)
	if err != nil {
		return GeofenceArea{}, fmt.Errorf("get geofence: %w", err)
	}

	area.Geometry = json.RawMessage(geometryText)
	return area, nil
}

func ListRecentTruckGeofenceEvents(ctx context.Context, pool *pgxpool.Pool, geofenceID string, limit int32) ([]TruckGeofenceEvent, error) {
	const query = `
SELECT
	id,
	truck_id,
	driver_name,
	geofence_id,
	geofence_name,
	event_type,
	latitude,
	longitude,
	occurred_at
FROM truck_geofence_events
WHERE geofence_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT $2
`

	rows, err := pool.Query(ctx, query, geofenceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent truck geofence events: %w", err)
	}
	defer rows.Close()

	events := make([]TruckGeofenceEvent, 0, limit)
	for rows.Next() {
		var event TruckGeofenceEvent
		if err := rows.Scan(
			&event.ID,
			&event.TruckID,
			&event.DriverName,
			&event.GeofenceID,
			&event.GeofenceName,
			&event.EventType,
			&event.Latitude,
			&event.Longitude,
			&event.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan truck geofence event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate truck geofence events: %w", err)
	}

	return events, nil
}

func ListTruckGeofenceEventsSince(ctx context.Context, pool *pgxpool.Pool, geofenceID string, since time.Time, lastID int64, limit int32) ([]TruckGeofenceEvent, error) {
	const query = `
SELECT
	id,
	truck_id,
	driver_name,
	geofence_id,
	geofence_name,
	event_type,
	latitude,
	longitude,
	occurred_at
FROM truck_geofence_events
WHERE geofence_id = $1
	AND (
		occurred_at > $2
		OR (occurred_at = $2 AND id > $3)
	)
ORDER BY occurred_at ASC, id ASC
LIMIT $4
`

	rows, err := pool.Query(ctx, query, geofenceID, since, lastID, limit)
	if err != nil {
		return nil, fmt.Errorf("list truck geofence events since cursor: %w", err)
	}
	defer rows.Close()

	events := make([]TruckGeofenceEvent, 0, limit)
	for rows.Next() {
		var event TruckGeofenceEvent
		if err := rows.Scan(
			&event.ID,
			&event.TruckID,
			&event.DriverName,
			&event.GeofenceID,
			&event.GeofenceName,
			&event.EventType,
			&event.Latitude,
			&event.Longitude,
			&event.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan truck geofence event since cursor: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate truck geofence events since cursor: %w", err)
	}

	return events, nil
}

func TruncateStores(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE TABLE stores`)
	if err != nil {
		return fmt.Errorf("truncate stores: %w", err)
	}

	return nil
}

func ImportCSV(ctx context.Context, pool *pgxpool.Pool, csvPath string) (int64, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, fmt.Errorf("open csv: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close csv: %w", closeErr)
		}
	}()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read csv header: %w", err)
	}

	columns, err := newCSVColumnSet(header)
	if err != nil {
		return 0, err
	}

	rows := make([][]any, 0, 1024)
	lineNumber := 1
	for {
		lineNumber++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("read csv record: %w", err)
		}

		storeNumber, err := columns.requiredValue(record, lineNumber, "Store Number")
		if err != nil {
			log.Printf("skipping %v", err)
			continue
		}

		latitudeText, err := columns.requiredValue(record, lineNumber, "Latitude")
		if err != nil {
			log.Printf("skipping %v", err)
			continue
		}

		latitude, err := strconv.ParseFloat(latitudeText, 64)
		if err != nil {
			log.Printf("skipping line %d: parse latitude for store %q: %v", lineNumber, storeNumber, err)
			continue
		}

		longitudeText, err := columns.requiredValue(record, lineNumber, "Longitude")
		if err != nil {
			log.Printf("skipping %v", err)
			continue
		}

		longitude, err := strconv.ParseFloat(longitudeText, 64)
		if err != nil {
			log.Printf("skipping line %d: parse longitude for store %q: %v", lineNumber, storeNumber, err)
			continue
		}

		timezoneValue := columns.optionalValue(record, "Timezone")

		rows = append(rows, []any{
			storeNumber,
			columns.optionalValue(record, "Country"),
			nullString(columns.optionalValue(record, "Ownership Type")),
			nil,
			nullString(columns.optionalValue(record, "Store Name")),
			latitude,
			longitude,
			nullString(columns.optionalValue(record, "Street Address")),
			nil,
			nil,
			nullString(columns.optionalValue(record, "City")),
			nullString(columns.optionalValue(record, "State/Province")),
			nullString(columns.optionalValue(record, "Postcode")),
			nullInt(parseTimezoneOffsetMinutes(timezoneValue)),
			nil,
			nullString(extractOlsonTimezone(timezoneValue)),
		})
	}

	count, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"stores"},
		[]string{
			"store_number",
			"country_code",
			"ownership_type_code",
			"schedule",
			"slug",
			"latitude",
			"longitude",
			"street_address_line1",
			"street_address_line2",
			"street_address_line3",
			"city",
			"country_subdivision_code",
			"postal_code",
			"current_time_offset",
			"windows_time_zone_id",
			"olson_time_zone_id",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("copy rows into stores: %w", err)
	}

	return count, nil
}

func Nearby(ctx context.Context, pool *pgxpool.Pool, latitude float64, longitude float64, radiusMeters float64, limit int32) ([]NearbyStore, error) {
	const query = `
SELECT
	store_number,
	country_code,
	COALESCE(city, ''),
	COALESCE(street_address_line1, ''),
	latitude,
	longitude,
	ROUND(ST_Distance(geog, ST_Point($2, $1, 4326)::geography))::BIGINT AS distance_meters
FROM stores
WHERE ST_DWithin(geog, ST_Point($2, $1, 4326)::geography, $3)
ORDER BY ST_Distance(geog, ST_Point($2, $1, 4326)::geography)
LIMIT $4
`

	rows, err := pool.Query(ctx, query, latitude, longitude, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("query nearby stores: %w", err)
	}
	defer rows.Close()

	stores := make([]NearbyStore, 0, limit)
	for rows.Next() {
		var store NearbyStore
		if err := rows.Scan(
			&store.StoreNumber,
			&store.CountryCode,
			&store.City,
			&store.StreetAddress,
			&store.Latitude,
			&store.Longitude,
			&store.DistanceMeters,
		); err != nil {
			return nil, fmt.Errorf("scan nearby store: %w", err)
		}
		stores = append(stores, store)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nearby stores: %w", err)
	}

	return stores, nil
}

func ClusterByCountry(ctx context.Context, pool *pgxpool.Pool, countryCode string, clusters int32) ([]ClusterResult, error) {
	const query = `
WITH clustered AS (
	SELECT
		ST_ClusterKMeans(location, $2) OVER () AS cluster_id,
		location
	FROM stores
	WHERE country_code = $1
)
SELECT
	cluster_id,
	COUNT(*)::INT,
	ROUND(ST_Y(ST_Centroid(ST_Collect(location)))::numeric, 5)::DOUBLE PRECISION AS center_latitude,
	ROUND(ST_X(ST_Centroid(ST_Collect(location)))::numeric, 5)::DOUBLE PRECISION AS center_longitude
FROM clustered
GROUP BY cluster_id
ORDER BY COUNT(*) DESC, cluster_id
`

	rows, err := pool.Query(ctx, query, countryCode, clusters)
	if err != nil {
		return nil, fmt.Errorf("query clusters: %w", err)
	}
	defer rows.Close()

	results := make([]ClusterResult, 0, clusters)
	for rows.Next() {
		var result ClusterResult
		if err := rows.Scan(
			&result.ClusterID,
			&result.StoreCount,
			&result.CenterLatitude,
			&result.CenterLongitude,
		); err != nil {
			return nil, fmt.Errorf("scan cluster result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clusters: %w", err)
	}

	return results, nil
}

func StoresAlongCorridor(ctx context.Context, pool *pgxpool.Pool, fromLatitude float64, fromLongitude float64, toLatitude float64, toLongitude float64, distanceMeters float64, limit int32) ([]CorridorStore, error) {
	const query = `
WITH route AS (
	SELECT ST_MakeLine(
		ST_Point($2, $1, 4326),
		ST_Point($4, $3, 4326)
	)::geography AS geog
)
SELECT
	s.store_number,
	s.country_code,
	COALESCE(s.city, ''),
	COALESCE(s.street_address_line1, ''),
	s.latitude,
	s.longitude,
	ROUND(ST_Distance(s.geog, route.geog))::BIGINT AS distance_to_route_meters
FROM stores s
CROSS JOIN route
WHERE ST_DWithin(s.geog, route.geog, $5)
ORDER BY ST_Distance(s.geog, route.geog), s.store_number
LIMIT $6
`

	rows, err := pool.Query(ctx, query, fromLatitude, fromLongitude, toLatitude, toLongitude, distanceMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("query corridor stores: %w", err)
	}
	defer rows.Close()

	stores := make([]CorridorStore, 0, limit)
	for rows.Next() {
		var store CorridorStore
		if err := rows.Scan(
			&store.StoreNumber,
			&store.CountryCode,
			&store.City,
			&store.StreetAddress,
			&store.Latitude,
			&store.Longitude,
			&store.DistanceToRouteMeters,
		); err != nil {
			return nil, fmt.Errorf("scan corridor store: %w", err)
		}
		stores = append(stores, store)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate corridor stores: %w", err)
	}

	return stores, nil
}

func ListCountries(ctx context.Context, pool *pgxpool.Pool, limit int32) ([]CountrySummary, error) {
	const query = `
SELECT country_code, COUNT(*)::INT AS store_count
FROM stores
GROUP BY country_code
ORDER BY COUNT(*) DESC, country_code
LIMIT $1
`

	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query countries: %w", err)
	}
	defer rows.Close()

	countries := make([]CountrySummary, 0, limit)
	for rows.Next() {
		var country CountrySummary
		if err := rows.Scan(&country.CountryCode, &country.StoreCount); err != nil {
			return nil, fmt.Errorf("scan country summary: %w", err)
		}
		countries = append(countries, country)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate country summaries: %w", err)
	}

	return countries, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullInt(value string) any {
	if value == "" {
		return nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}

	return parsed
}

type csvColumnSet map[string]int

func newCSVColumnSet(header []string) (csvColumnSet, error) {
	columns := make(csvColumnSet, len(header))
	for index, name := range header {
		columns[strings.TrimSpace(name)] = index
	}

	requiredColumns := []string{
		"Store Number",
		"Store Name",
		"Ownership Type",
		"Street Address",
		"City",
		"State/Province",
		"Country",
		"Postcode",
		"Timezone",
		"Longitude",
		"Latitude",
	}

	for _, name := range requiredColumns {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("csv header missing required column %q", name)
		}
	}

	return columns, nil
}

func (columns csvColumnSet) requiredValue(record []string, lineNumber int, name string) (string, error) {
	index, ok := columns[name]
	if !ok {
		return "", fmt.Errorf("csv header missing required column %q", name)
	}
	if index >= len(record) {
		return "", fmt.Errorf("csv line %d missing value for %q", lineNumber, name)
	}

	value := strings.TrimSpace(record[index])
	if value == "" {
		return "", fmt.Errorf("csv line %d has empty value for %q", lineNumber, name)
	}

	return value, nil
}

func (columns csvColumnSet) optionalValue(record []string, name string) string {
	index, ok := columns[name]
	if !ok || index >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[index])
}

func parseTimezoneOffsetMinutes(value string) string {
	if !strings.HasPrefix(value, "GMT") {
		return ""
	}

	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}

	offset := strings.TrimPrefix(parts[0], "GMT")
	separator := strings.Index(offset, ":")
	if separator <= 1 || separator+2 >= len(offset) {
		return ""
	}

	sign := 1
	if offset[0] == '-' {
		sign = -1
	} else if offset[0] != '+' {
		return ""
	}

	hours, err := strconv.Atoi(offset[1:separator])
	if err != nil {
		return ""
	}

	minutes, err := strconv.Atoi(offset[separator+1:])
	if err != nil {
		return ""
	}

	return strconv.Itoa(sign * ((hours * 60) + minutes))
}

func extractOlsonTimezone(value string) string {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-1]
}
