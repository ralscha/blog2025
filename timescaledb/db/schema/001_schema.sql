CREATE TABLE IF NOT EXISTS sensor_data (
    time        TIMESTAMPTZ NOT NULL,
    sensor_id   TEXT NOT NULL,
    temperature DOUBLE PRECISION,
    humidity    DOUBLE PRECISION,
    PRIMARY KEY (time, sensor_id)
);

CREATE TABLE IF NOT EXISTS stock_prices (
    time   TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    price  DOUBLE PRECISION,
    volume INTEGER,
    PRIMARY KEY (time, symbol)
);

-- sqlc schema placeholder so query typing knows the continuous aggregate columns.
CREATE VIEW sensor_hourly AS
SELECT
    now()::timestamptz AS bucket,
    ''::text AS sensor_id,
    0::double precision AS avg_temp,
    0::double precision AS min_temp,
    0::double precision AS max_temp,
    0::bigint AS sample_count;
