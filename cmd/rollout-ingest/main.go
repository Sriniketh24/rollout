package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Sriniketh24/rollout/internal/config"
	"github.com/Sriniketh24/rollout/internal/health"
	"github.com/Sriniketh24/rollout/internal/models"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/nats-io/nats.go"
)

const ingestVersion = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	batchSize := envIntOr("BATCH_SIZE", 1000)
	flushInterval := envDurationOr("FLUSH_INTERVAL", 5*time.Second)

	h := health.New("rollout-ingest", ingestVersion)

	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.User,
			Password: cfg.ClickHouse.Password,
		},
	})
	if err != nil {
		logger.Error("clickhouse connection failed", "error", err)
		os.Exit(1)
	}

	h.Register("clickhouse", func(ctx context.Context) error {
		return chConn.Ping(ctx)
	})

	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		logger.Error("nats connection failed", "error", err)
		os.Exit(1)
	}

	h.Register("nats", func(ctx context.Context) error {
		if !nc.IsConnected() {
			return fmt.Errorf("nats disconnected")
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exposureBuf := make(chan models.ExposureEvent, batchSize*2)
	conversionBuf := make(chan models.ConversionEvent, batchSize*2)

	nc.Subscribe("rollout.*.*.exposures", func(msg *nats.Msg) {
		var event models.ExposureEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("exposure unmarshal failed", "error", err)
			return
		}
		select {
		case exposureBuf <- event:
		default:
			logger.Warn("exposure buffer full, dropping event")
		}
	})

	nc.Subscribe("rollout.*.*.conversions", func(msg *nats.Msg) {
		var event models.ConversionEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("conversion unmarshal failed", "error", err)
			return
		}
		select {
		case conversionBuf <- event:
		default:
			logger.Warn("conversion buffer full, dropping event")
		}
	})

	go flushExposures(ctx, chConn, exposureBuf, batchSize, flushInterval, logger)
	go flushConversions(ctx, chConn, conversionBuf, batchSize, flushInterval, logger)

	logger.Info("ingest worker started",
		"batch_size", batchSize,
		"flush_interval", flushInterval,
		"clickhouse", fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down ingest worker")
	cancel()
	nc.Drain()
	chConn.Close()
	logger.Info("ingest worker stopped")

	_ = h
}

func flushExposures(ctx context.Context, conn clickhouse.Conn, buf chan models.ExposureEvent, batchSize int, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([]models.ExposureEvent, 0, batchSize)

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				writeExposureBatch(context.Background(), conn, batch, logger)
			}
			return
		case event := <-buf:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				writeExposureBatch(ctx, conn, batch, logger)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				writeExposureBatch(ctx, conn, batch, logger)
				batch = batch[:0]
			}
		}
	}
}

func flushConversions(ctx context.Context, conn clickhouse.Conn, buf chan models.ConversionEvent, batchSize int, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batch := make([]models.ConversionEvent, 0, batchSize)

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				writeConversionBatch(context.Background(), conn, batch, logger)
			}
			return
		case event := <-buf:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				writeConversionBatch(ctx, conn, batch, logger)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				writeConversionBatch(ctx, conn, batch, logger)
				batch = batch[:0]
			}
		}
	}
}

func writeExposureBatch(ctx context.Context, conn clickhouse.Conn, events []models.ExposureEvent, logger *slog.Logger) {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO exposures (flag_key, environment_id, user_key, variation, reason, timestamp)")
	if err != nil {
		logger.Error("exposure batch prepare failed", "error", err)
		return
	}

	for _, e := range events {
		if err := batch.Append(e.FlagKey, e.EnvironmentID, e.UserKey, string(e.Variation), string(e.Reason), e.Timestamp); err != nil {
			logger.Error("exposure append failed", "error", err)
		}
	}

	if err := batch.Send(); err != nil {
		logger.Error("exposure batch send failed", "error", err, "count", len(events))
	} else {
		logger.Info("flushed exposures", "count", len(events))
	}
}

func writeConversionBatch(ctx context.Context, conn clickhouse.Conn, events []models.ConversionEvent, logger *slog.Logger) {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO conversions (experiment_id, environment_id, user_key, metric_key, value, timestamp)")
	if err != nil {
		logger.Error("conversion batch prepare failed", "error", err)
		return
	}

	for _, e := range events {
		if err := batch.Append(e.ExperimentID, e.EnvironmentID, e.UserKey, e.MetricKey, e.Value, e.Timestamp); err != nil {
			logger.Error("conversion append failed", "error", err)
		}
	}

	if err := batch.Send(); err != nil {
		logger.Error("conversion batch send failed", "error", err, "count", len(events))
	} else {
		logger.Info("flushed conversions", "count", len(events))
	}
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
