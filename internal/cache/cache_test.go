package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setup(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := New(client, 5*time.Minute)
	return c, mr
}

func sampleFlag() *models.FlagEnvironment {
	return &models.FlagEnvironment{
		FlagID:        "flag-1",
		EnvironmentID: "env-1",
		Enabled:       true,
		Version:       42,
		OffVariation:  json.RawMessage(`false`),
		UpdatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestGetFlag_Miss(t *testing.T) {
	c, _ := setup(t)
	fe, err := c.GetFlag(context.Background(), "proj", "env", "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fe != nil {
		t.Fatal("expected nil on cache miss")
	}
}

func TestSetAndGetFlag(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()
	fe := sampleFlag()

	if err := c.SetFlag(ctx, "proj", "env", "my-flag", fe); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}

	got, err := c.GetFlag(ctx, "proj", "env", "my-flag")
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}
	if got == nil {
		t.Fatal("expected cached value, got nil")
	}
	if got.FlagID != fe.FlagID {
		t.Errorf("FlagID = %q, want %q", got.FlagID, fe.FlagID)
	}
	if got.Version != fe.Version {
		t.Errorf("Version = %d, want %d", got.Version, fe.Version)
	}
	if got.Enabled != fe.Enabled {
		t.Errorf("Enabled = %v, want %v", got.Enabled, fe.Enabled)
	}
}

func TestInvalidateFlag(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()

	if err := c.SetFlag(ctx, "proj", "env", "flag-a", sampleFlag()); err != nil {
		t.Fatal(err)
	}
	if err := c.InvalidateFlag(ctx, "proj", "env", "flag-a"); err != nil {
		t.Fatal(err)
	}

	got, err := c.GetFlag(ctx, "proj", "env", "flag-a")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil after invalidation")
	}
}

func TestInvalidateEnvironment(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()

	// Populate several keys under the same environment.
	if err := c.SetFlag(ctx, "proj", "env", "flag-a", sampleFlag()); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFlag(ctx, "proj", "env", "flag-b", sampleFlag()); err != nil {
		t.Fatal(err)
	}
	if err := c.SetEnvironmentFlags(ctx, "proj", "env", []byte(`{"all":"flags"}`)); err != nil {
		t.Fatal(err)
	}
	// A flag in a different environment should survive.
	if err := c.SetFlag(ctx, "proj", "other-env", "flag-c", sampleFlag()); err != nil {
		t.Fatal(err)
	}

	if err := c.InvalidateEnvironment(ctx, "proj", "env"); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"flag-a", "flag-b"} {
		got, err := c.GetFlag(ctx, "proj", "env", key)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("expected %s to be invalidated", key)
		}
	}

	bulk, err := c.GetEnvironmentFlags(ctx, "proj", "env")
	if err != nil {
		t.Fatal(err)
	}
	if bulk != nil {
		t.Error("expected env flags to be invalidated")
	}

	// The other environment should be untouched.
	survivor, err := c.GetFlag(ctx, "proj", "other-env", "flag-c")
	if err != nil {
		t.Fatal(err)
	}
	if survivor == nil {
		t.Error("flag in other env should not be invalidated")
	}
}

func TestEnvironmentFlags_BulkRoundTrip(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()

	payload := []byte(`[{"flag_id":"f1","enabled":true},{"flag_id":"f2","enabled":false}]`)
	if err := c.SetEnvironmentFlags(ctx, "proj", "env", payload); err != nil {
		t.Fatal(err)
	}

	got, err := c.GetEnvironmentFlags(ctx, "proj", "env")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("got %s, want %s", got, payload)
	}
}

func TestGetEnvironmentFlags_Miss(t *testing.T) {
	c, _ := setup(t)
	got, err := c.GetEnvironmentFlags(context.Background(), "proj", "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil on miss")
	}
}

func TestGetAllFlagVersions(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()

	fe1 := sampleFlag()
	fe1.Version = 10
	fe2 := sampleFlag()
	fe2.Version = 20

	if err := c.SetFlag(ctx, "proj", "env", "flag-x", fe1); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFlag(ctx, "proj", "env", "flag-y", fe2); err != nil {
		t.Fatal(err)
	}
	// Different environment, should not appear.
	if err := c.SetFlag(ctx, "proj", "other", "flag-z", sampleFlag()); err != nil {
		t.Fatal(err)
	}

	versions, err := c.GetAllFlagVersions(ctx, "proj", "env")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(versions))
	}
	if versions["flag-x"] != 10 {
		t.Errorf("flag-x version = %d, want 10", versions["flag-x"])
	}
	if versions["flag-y"] != 20 {
		t.Errorf("flag-y version = %d, want 20", versions["flag-y"])
	}
}

func TestTTLExpiry(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := New(client, 2*time.Second)
	ctx := context.Background()

	if err := c.SetFlag(ctx, "proj", "env", "ttl-flag", sampleFlag()); err != nil {
		t.Fatal(err)
	}

	// Fast-forward miniredis time past the TTL.
	mr.FastForward(3 * time.Second)

	got, err := c.GetFlag(ctx, "proj", "env", "ttl-flag")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil after TTL expiry")
	}
}
