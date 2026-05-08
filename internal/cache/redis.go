package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client with typed methods for flag configuration caching.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// New creates a Cache backed by the given Redis client. All entries expire
// after ttl unless explicitly invalidated first.
func New(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

// ── key helpers ──────────────────────────────────────────────────────────────

func flagKey(projectID, envID, flagKey string) string {
	return fmt.Sprintf("rollout:%s:%s:flag:%s", projectID, envID, flagKey)
}

func envFlagsKey(projectID, envID string) string {
	return fmt.Sprintf("rollout:%s:%s:flags", projectID, envID)
}

func envPrefix(projectID, envID string) string {
	return fmt.Sprintf("rollout:%s:%s:", projectID, envID)
}

// ── single-flag operations ───────────────────────────────────────────────────

// GetFlag returns the cached FlagEnvironment for a specific flag in an
// environment. On a cache miss it returns (nil, nil).
func (c *Cache) GetFlag(ctx context.Context, projectID, envID, key string) (*models.FlagEnvironment, error) {
	data, err := c.client.Get(ctx, flagKey(projectID, envID, key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get flag: %w", err)
	}

	var fe models.FlagEnvironment
	if err := json.Unmarshal(data, &fe); err != nil {
		return nil, fmt.Errorf("cache unmarshal flag: %w", err)
	}
	return &fe, nil
}

// SetFlag stores a FlagEnvironment in the cache.
func (c *Cache) SetFlag(ctx context.Context, projectID, envID, key string, fe *models.FlagEnvironment) error {
	data, err := json.Marshal(fe)
	if err != nil {
		return fmt.Errorf("cache marshal flag: %w", err)
	}
	if err := c.client.Set(ctx, flagKey(projectID, envID, key), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("cache set flag: %w", err)
	}
	return nil
}

// InvalidateFlag removes a single flag entry from the cache.
func (c *Cache) InvalidateFlag(ctx context.Context, projectID, envID, key string) error {
	if err := c.client.Del(ctx, flagKey(projectID, envID, key)).Err(); err != nil {
		return fmt.Errorf("cache invalidate flag: %w", err)
	}
	return nil
}

// InvalidateEnvironment removes all cached entries for a given environment,
// including individual flag keys and the bulk flags payload.
func (c *Cache) InvalidateEnvironment(ctx context.Context, projectID, envID string) error {
	prefix := envPrefix(projectID, envID)
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan for invalidation: %w", err)
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache delete keys: %w", err)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// ── bulk environment flags ───────────────────────────────────────────────────

// GetEnvironmentFlags returns the raw JSON blob containing the full flag
// ruleset for an environment. On a cache miss it returns (nil, nil).
func (c *Cache) GetEnvironmentFlags(ctx context.Context, projectID, envID string) ([]byte, error) {
	data, err := c.client.Get(ctx, envFlagsKey(projectID, envID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get env flags: %w", err)
	}
	return data, nil
}

// SetEnvironmentFlags stores the full flag ruleset for an environment as a raw
// JSON blob.
func (c *Cache) SetEnvironmentFlags(ctx context.Context, projectID, envID string, data []byte) error {
	if err := c.client.Set(ctx, envFlagsKey(projectID, envID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("cache set env flags: %w", err)
	}
	return nil
}

// ── version tracking ─────────────────────────────────────────────────────────

// GetAllFlagVersions scans every individual flag key cached for the given
// environment and returns a map of flagKey → version. This is useful for
// comparing against the source of truth to decide which entries need
// invalidation.
func (c *Cache) GetAllFlagVersions(ctx context.Context, projectID, envID string) (map[string]int64, error) {
	prefix := envPrefix(projectID, envID) + "flag:"
	versions := make(map[string]int64)

	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("cache scan versions: %w", err)
		}
		for _, k := range keys {
			data, err := c.client.Get(ctx, k).Bytes()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("cache get for version: %w", err)
			}
			var fe models.FlagEnvironment
			if err := json.Unmarshal(data, &fe); err != nil {
				return nil, fmt.Errorf("cache unmarshal for version: %w", err)
			}
			// Extract the flagKey from the Redis key.
			// Key format: rollout:{projectID}:{envID}:flag:{flagKey}
			parts := strings.SplitN(k, ":flag:", 2)
			if len(parts) == 2 {
				versions[parts[1]] = fe.Version
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return versions, nil
}
