-- name: EnableTimescaleExtension :exec
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- name: CreateSensorDataTable :exec
CREATE TABLE IF NOT EXISTS sensor_data (
    time        TIMESTAMPTZ NOT NULL,
    sensor_id   TEXT NOT NULL,
    temperature DOUBLE PRECISION,
    humidity    DOUBLE PRECISION,
    PRIMARY KEY (time, sensor_id)
);

-- name: CreateSensorDataIndex :exec
CREATE INDEX IF NOT EXISTS idx_sensor_data_sensor_time
ON sensor_data (sensor_id, time DESC);

-- name: CreateSensorDataHypertable :exec
SELECT create_hypertable('sensor_data', by_range('time'), if_not_exists => TRUE);

-- name: SetSensorDataChunkInterval :exec
SELECT set_chunk_time_interval('sensor_data', INTERVAL '1 day');

-- name: InsertSampleSensorData :exec
INSERT INTO sensor_data (time, sensor_id, temperature, humidity)
SELECT
    ts,
    'sensor_' || ((random() * 4)::int + 1),
    18 + random() * 12,
    35 + random() * 25
FROM generate_series(
    now() - interval '24 hours',
    now(),
    interval '10 seconds'
) ts;

-- name: CreateSensorHourlyCagg :exec
CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    sensor_id,
    avg(temperature) AS avg_temp,
    min(temperature) AS min_temp,
    max(temperature) AS max_temp,
    avg(humidity) AS avg_humidity,
    count(*) AS sample_count
FROM sensor_data
GROUP BY bucket, sensor_id;

-- name: AddSensorHourlyCaggPolicy :exec
DO $$
BEGIN
    PERFORM add_continuous_aggregate_policy(
        'sensor_hourly',
        start_offset => interval '2 days',
        end_offset => interval '10 minutes',
        schedule_interval => interval '10 minutes'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN invalid_parameter_value THEN
        IF SQLERRM LIKE '%already exists%' THEN
            NULL;
        ELSE
            RAISE;
        END IF;
END;
$$;

-- name: SetSensorDataColumnstore :exec
ALTER TABLE sensor_data
SET (
    timescaledb.enable_columnstore = true,
    timescaledb.compress_segmentby = 'sensor_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- name: AddSensorDataColumnstorePolicy :exec
CALL add_columnstore_policy(
    'sensor_data',
    after => INTERVAL '7 days',
    if_not_exists => true
);

-- name: AddSensorDataRetentionPolicy :exec
DO $$
BEGIN
    PERFORM add_retention_policy('sensor_data', interval '30 days');
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN invalid_parameter_value THEN
        IF SQLERRM LIKE '%already exists%' THEN
            NULL;
        ELSE
            RAISE;
        END IF;
END;
$$;

-- name: AnalyzeSensorData :exec
ANALYZE sensor_data;

-- name: GetSensorStatistics :many
SELECT sensor_id,
       COUNT(*) AS count,
       AVG(temperature) AS avg_temp,
       AVG(humidity) AS avg_humidity
FROM sensor_data
GROUP BY sensor_id
ORDER BY sensor_id;

-- name: GetHourlyAverageTemperatureLast24h :many
SELECT time_bucket('1 hour', time)::timestamptz AS hour,
       AVG(temperature) AS avg_temp
FROM sensor_data
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour DESC
LIMIT 10;

-- name: GetLatestReadingsPerSensor :many
SELECT DISTINCT ON (sensor_id)
       sensor_id,
       time,
       temperature,
       humidity
FROM sensor_data
ORDER BY sensor_id, time DESC;

-- name: GetFiveMinuteTemperatureStatsLastHour :many
SELECT time_bucket('5 minutes', time)::timestamptz AS bucket,
       AVG(temperature) AS avg_temp,
       COALESCE(MIN(temperature), 0)::float8 AS min_temp,
       COALESCE(MAX(temperature), 0)::float8 AS max_temp
FROM sensor_data
WHERE time > NOW() - INTERVAL '1 hour'
GROUP BY bucket
ORDER BY bucket DESC
LIMIT 5;

-- name: GetGapfilledSensorTemperature :many
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
LIMIT 5;

-- name: GetSensorHourlyAggregateLast7Days :many
SELECT
    bucket,
    sensor_id,
    round(avg_temp::numeric, 2)::float8 AS avg_temp,
    round(min_temp::numeric, 2)::float8 AS min_temp,
    round(max_temp::numeric, 2)::float8 AS max_temp,
    sample_count
FROM sensor_hourly
WHERE bucket > now() - interval '7 days'
ORDER BY bucket DESC, sensor_id
LIMIT 10;

-- name: CreateStockPricesTable :exec
CREATE TABLE IF NOT EXISTS stock_prices (
    time   TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    price  DOUBLE PRECISION,
    volume INTEGER
);

-- name: CreateStockPricesIndex :exec
CREATE INDEX IF NOT EXISTS idx_stock_prices_symbol_time
ON stock_prices (symbol, time DESC);

-- name: CreateStockPricesHypertable :exec
SELECT create_hypertable('stock_prices', by_range('time'), if_not_exists => TRUE);

-- name: InsertStockPrice :exec
INSERT INTO stock_prices (time, symbol, price, volume)
VALUES ($1, $2, $3, $4);

-- name: GetStockPriceSummaryBySymbol :many
SELECT symbol,
       COUNT(*) AS count,
       AVG(price) AS avg_price,
    COALESCE(MIN(price), 0)::float8 AS min_price,
    COALESCE(MAX(price), 0)::float8 AS max_price,
       SUM(volume) AS total_volume
FROM stock_prices
GROUP BY symbol
ORDER BY symbol;

-- name: GetStockPriceTrendsLast6Hours :many
SELECT symbol,
    time_bucket('1 hour', time)::timestamptz AS hour,
       AVG(price) AS avg_price
FROM stock_prices
WHERE time > NOW() - INTERVAL '6 hours'
GROUP BY symbol, hour
ORDER BY symbol, hour DESC
LIMIT 15;

-- name: DropSensorHourlyView :exec
DROP MATERIALIZED VIEW IF EXISTS sensor_hourly CASCADE;

-- name: DropSensorDataTable :exec
DROP TABLE IF EXISTS sensor_data CASCADE;

-- name: DropStockPricesTable :exec
DROP TABLE IF EXISTS stock_prices CASCADE;
