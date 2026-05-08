-- ClickHouse schema for Rollout analytics
-- Exposure events table
CREATE TABLE IF NOT EXISTS exposures (
    flag_key       String,
    environment_id String,
    user_key       String,
    variation      String,
    reason         String,
    timestamp      DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (flag_key, environment_id, timestamp);

-- Conversion events table
CREATE TABLE IF NOT EXISTS conversions (
    experiment_id  String,
    environment_id String,
    user_key       String,
    metric_key     String,
    value          Float64,
    timestamp      DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (experiment_id, environment_id, timestamp);

-- Materialized view: hourly exposure aggregations
CREATE TABLE IF NOT EXISTS exposures_hourly (
    flag_key       String,
    environment_id String,
    variation      String,
    hour           DateTime,
    count          UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (flag_key, environment_id, variation, hour);

CREATE MATERIALIZED VIEW IF NOT EXISTS exposures_hourly_mv
TO exposures_hourly
AS
SELECT
    flag_key,
    environment_id,
    variation,
    toStartOfHour(timestamp) AS hour,
    count() AS count
FROM exposures
GROUP BY flag_key, environment_id, variation, hour;

-- Materialized view: hourly conversion aggregations
CREATE TABLE IF NOT EXISTS conversions_hourly (
    experiment_id  String,
    environment_id String,
    metric_key     String,
    hour           DateTime,
    count          UInt64,
    total_value    Float64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (experiment_id, environment_id, metric_key, hour);

CREATE MATERIALIZED VIEW IF NOT EXISTS conversions_hourly_mv
TO conversions_hourly
AS
SELECT
    experiment_id,
    environment_id,
    metric_key,
    toStartOfHour(timestamp) AS hour,
    count()  AS count,
    sum(value) AS total_value
FROM conversions
GROUP BY experiment_id, environment_id, metric_key, hour;
