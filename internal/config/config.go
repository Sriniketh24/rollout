package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server    ServerConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	ClickHouse ClickHouseConfig
	NATS      NATSConfig
	Auth      AuthConfig
	Chaos     ChaosConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type NATSConfig struct {
	URL string
}

type AuthConfig struct {
	JWTSecret     string
	TokenExpiry   time.Duration
	APIKeyHeader  string
}

type ChaosConfig struct {
	Enabled          bool
	FailureRate      float64
	LatencyInjection time.Duration
	StaleCacheRate   float64
}

func (c *PostgresConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.Database + "?sslmode=" + c.SSLMode
}

func (c *ClickHouseConfig) DSN() string {
	return "clickhouse://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.Database
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            envOr("SERVER_HOST", "0.0.0.0"),
			Port:            envIntOr("SERVER_PORT", 8080),
			ReadTimeout:     envDurationOr("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    envDurationOr("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: envDurationOr("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Postgres: PostgresConfig{
			Host:     envOr("POSTGRES_HOST", "localhost"),
			Port:     envIntOr("POSTGRES_PORT", 5432),
			User:     envOr("POSTGRES_USER", "rollout"),
			Password: envOr("POSTGRES_PASSWORD", "rollout"),
			Database: envOr("POSTGRES_DB", "rollout"),
			SSLMode:  envOr("POSTGRES_SSLMODE", "disable"),
			MaxConns: envIntOr("POSTGRES_MAX_CONNS", 25),
		},
		Redis: RedisConfig{
			Host:     envOr("REDIS_HOST", "localhost"),
			Port:     envIntOr("REDIS_PORT", 6379),
			Password: envOr("REDIS_PASSWORD", ""),
			DB:       envIntOr("REDIS_DB", 0),
		},
		ClickHouse: ClickHouseConfig{
			Host:     envOr("CLICKHOUSE_HOST", "localhost"),
			Port:     envIntOr("CLICKHOUSE_PORT", 9000),
			User:     envOr("CLICKHOUSE_USER", "default"),
			Password: envOr("CLICKHOUSE_PASSWORD", ""),
			Database: envOr("CLICKHOUSE_DB", "rollout"),
		},
		NATS: NATSConfig{
			URL: envOr("NATS_URL", "nats://localhost:4222"),
		},
		Auth: AuthConfig{
			JWTSecret:    envOr("JWT_SECRET", "dev-secret-change-in-production"),
			TokenExpiry:  envDurationOr("TOKEN_EXPIRY", 24*time.Hour),
			APIKeyHeader: envOr("API_KEY_HEADER", "X-Rollout-Key"),
		},
		Chaos: ChaosConfig{
			Enabled:          envBoolOr("CHAOS_ENABLED", false),
			FailureRate:      envFloatOr("CHAOS_FAILURE_RATE", 0.0),
			LatencyInjection: envDurationOr("CHAOS_LATENCY", 0),
			StaleCacheRate:   envFloatOr("CHAOS_STALE_CACHE_RATE", 0.0),
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envBoolOr(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envFloatOr(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
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
