package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Sriniketh24/rollout/internal/models"
)

// VariationCount holds exposure or conversion counts for a single variation.
type VariationCount struct {
	Variation string `json:"variation"`
	Count     int64  `json:"count"`
}

// ExperimentRawResult holds the raw per-variation data needed for Bayesian analysis.
type ExperimentRawResult struct {
	Variation   string  `json:"variation"`
	Exposures   int64   `json:"exposures"`
	Conversions int64   `json:"conversions"`
	TotalValue  float64 `json:"total_value"`
}

// TimeseriesPoint is a single data point in a timeseries.
type TimeseriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Variation string    `json:"variation"`
	Count     int64     `json:"count"`
}

// Analytics provides read/write access to ClickHouse analytics tables.
type Analytics struct {
	conn clickhouse.Conn
}

// New creates a new Analytics store backed by the given ClickHouse connection.
func New(conn clickhouse.Conn) *Analytics {
	return &Analytics{conn: conn}
}

// InsertExposures batch-inserts exposure events into ClickHouse.
func (a *Analytics) InsertExposures(ctx context.Context, events []models.ExposureEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := a.conn.PrepareBatch(ctx, "INSERT INTO exposures (flag_key, environment_id, user_key, variation, reason, timestamp)")
	if err != nil {
		return fmt.Errorf("prepare exposures batch: %w", err)
	}

	for _, e := range events {
		if err := batch.Append(
			e.FlagKey,
			e.EnvironmentID,
			e.UserKey,
			string(e.Variation),
			string(e.Reason),
			e.Timestamp,
		); err != nil {
			return fmt.Errorf("append exposure row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send exposures batch: %w", err)
	}
	return nil
}

// InsertConversions batch-inserts conversion events into ClickHouse.
func (a *Analytics) InsertConversions(ctx context.Context, events []models.ConversionEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := a.conn.PrepareBatch(ctx, "INSERT INTO conversions (experiment_id, environment_id, user_key, metric_key, value, timestamp)")
	if err != nil {
		return fmt.Errorf("prepare conversions batch: %w", err)
	}

	for _, e := range events {
		if err := batch.Append(
			e.ExperimentID,
			e.EnvironmentID,
			e.UserKey,
			e.MetricKey,
			e.Value,
			e.Timestamp,
		); err != nil {
			return fmt.Errorf("append conversion row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send conversions batch: %w", err)
	}
	return nil
}

// GetExposureCounts returns per-variation exposure counts for the given flag and
// environment within the specified time range.
func (a *Analytics) GetExposureCounts(
	ctx context.Context,
	flagKey string,
	envID string,
	startTime time.Time,
	endTime time.Time,
) ([]VariationCount, error) {
	query := `
		SELECT variation, count() AS cnt
		FROM exposures
		WHERE flag_key = @flagKey
		  AND environment_id = @envID
		  AND timestamp >= @start
		  AND timestamp <= @end
		GROUP BY variation
		ORDER BY cnt DESC
	`

	rows, err := a.conn.Query(ctx, query,
		clickhouse.Named("flagKey", flagKey),
		clickhouse.Named("envID", envID),
		clickhouse.Named("start", startTime),
		clickhouse.Named("end", endTime),
	)
	if err != nil {
		return nil, fmt.Errorf("query exposure counts: %w", err)
	}
	defer rows.Close()

	var results []VariationCount
	for rows.Next() {
		var vc VariationCount
		if err := rows.Scan(&vc.Variation, &vc.Count); err != nil {
			return nil, fmt.Errorf("scan exposure count row: %w", err)
		}
		results = append(results, vc)
	}
	return results, rows.Err()
}

// GetConversionCounts returns per-variation conversion counts for the given
// experiment and metric within the specified time range.
func (a *Analytics) GetConversionCounts(
	ctx context.Context,
	experimentID string,
	metricKey string,
	startTime time.Time,
	endTime time.Time,
) ([]VariationCount, error) {
	// Join conversions with exposures to attribute conversions to variations.
	query := `
		SELECT e.variation, count() AS cnt
		FROM conversions c
		INNER JOIN (
			SELECT DISTINCT user_key, variation
			FROM exposures
			WHERE flag_key IN (
				SELECT DISTINCT flag_key FROM exposures
				WHERE environment_id IN (
					SELECT DISTINCT environment_id FROM conversions WHERE experiment_id = @expID
				)
			)
		) e ON c.user_key = e.user_key
		WHERE c.experiment_id = @expID
		  AND c.metric_key = @metricKey
		  AND c.timestamp >= @start
		  AND c.timestamp <= @end
		GROUP BY e.variation
		ORDER BY cnt DESC
	`

	rows, err := a.conn.Query(ctx, query,
		clickhouse.Named("expID", experimentID),
		clickhouse.Named("metricKey", metricKey),
		clickhouse.Named("start", startTime),
		clickhouse.Named("end", endTime),
	)
	if err != nil {
		return nil, fmt.Errorf("query conversion counts: %w", err)
	}
	defer rows.Close()

	var results []VariationCount
	for rows.Next() {
		var vc VariationCount
		if err := rows.Scan(&vc.Variation, &vc.Count); err != nil {
			return nil, fmt.Errorf("scan conversion count row: %w", err)
		}
		results = append(results, vc)
	}
	return results, rows.Err()
}

// GetExperimentResults returns per-variation raw data needed for Bayesian
// analysis: exposure count, conversion count, and total conversion value.
func (a *Analytics) GetExperimentResults(
	ctx context.Context,
	experimentID string,
) ([]ExperimentRawResult, error) {
	query := `
		SELECT
			exp.variation,
			exp.exposures,
			COALESCE(conv.conversions, 0)  AS conversions,
			COALESCE(conv.total_value, 0)  AS total_value
		FROM (
			SELECT variation, count() AS exposures
			FROM exposures e
			INNER JOIN (
				SELECT DISTINCT environment_id
				FROM conversions
				WHERE experiment_id = @expID
			) env ON e.environment_id = env.environment_id
			GROUP BY variation
		) exp
		LEFT JOIN (
			SELECT e.variation, count() AS conversions, sum(c.value) AS total_value
			FROM conversions c
			INNER JOIN (
				SELECT DISTINCT user_key, variation
				FROM exposures
			) e ON c.user_key = e.user_key
			WHERE c.experiment_id = @expID
			GROUP BY e.variation
		) conv ON exp.variation = conv.variation
		ORDER BY exp.exposures DESC
	`

	rows, err := a.conn.Query(ctx, query,
		clickhouse.Named("expID", experimentID),
	)
	if err != nil {
		return nil, fmt.Errorf("query experiment results: %w", err)
	}
	defer rows.Close()

	var results []ExperimentRawResult
	for rows.Next() {
		var r ExperimentRawResult
		if err := rows.Scan(&r.Variation, &r.Exposures, &r.Conversions, &r.TotalValue); err != nil {
			return nil, fmt.Errorf("scan experiment result row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// GetFlagExposureTimeseries returns timeseries exposure data bucketed by the
// given granularity ("minute", "hour", "day", "week", "month").
func (a *Analytics) GetFlagExposureTimeseries(
	ctx context.Context,
	flagKey string,
	envID string,
	granularity string,
	startTime time.Time,
	endTime time.Time,
) ([]TimeseriesPoint, error) {
	truncFn, err := granularityToTruncFn(granularity)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT %s(timestamp) AS bucket, variation, count() AS cnt
		FROM exposures
		WHERE flag_key = @flagKey
		  AND environment_id = @envID
		  AND timestamp >= @start
		  AND timestamp <= @end
		GROUP BY bucket, variation
		ORDER BY bucket ASC, variation ASC
	`, truncFn)

	rows, err := a.conn.Query(ctx, query,
		clickhouse.Named("flagKey", flagKey),
		clickhouse.Named("envID", envID),
		clickhouse.Named("start", startTime),
		clickhouse.Named("end", endTime),
	)
	if err != nil {
		return nil, fmt.Errorf("query exposure timeseries: %w", err)
	}
	defer rows.Close()

	var points []TimeseriesPoint
	for rows.Next() {
		var p TimeseriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Variation, &p.Count); err != nil {
			return nil, fmt.Errorf("scan timeseries row: %w", err)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// granularityToTruncFn maps a human-readable granularity string to a ClickHouse
// date truncation function name.
func granularityToTruncFn(g string) (string, error) {
	switch g {
	case "minute":
		return "toStartOfMinute", nil
	case "hour":
		return "toStartOfHour", nil
	case "day":
		return "toStartOfDay", nil
	case "week":
		return "toStartOfWeek", nil
	case "month":
		return "toStartOfMonth", nil
	default:
		return "", fmt.Errorf("unsupported granularity %q: use minute, hour, day, week, or month", g)
	}
}
