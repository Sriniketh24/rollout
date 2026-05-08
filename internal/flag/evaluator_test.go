package flag

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
)

type mockStore struct {
	flags    map[string]*models.Flag
	flagEnvs map[string]*models.FlagEnvironment
}

func (m *mockStore) GetFlag(projectID, flagKey string) (*models.Flag, error) {
	f, ok := m.flags[projectID+":"+flagKey]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return f, nil
}

func (m *mockStore) GetFlagEnvironment(flagID, envID string) (*models.FlagEnvironment, error) {
	fe, ok := m.flagEnvs[flagID+":"+envID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return fe, nil
}

func (m *mockStore) GetAllFlagsForEnvironment(projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error) {
	return nil, nil, nil
}

func TestEvaluate_FlagOff(t *testing.T) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:dark-mode": {
				ID:        "flag-1",
				ProjectID: "proj",
				Key:       "dark-mode",
				Type:      models.FlagTypeBoolean,
				Enabled:   false,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"flag-1:prod": {
				FlagID:        "flag-1",
				EnvironmentID: "prod",
				Enabled:       true,
				OffVariation:  json.RawMessage(`false`),
				Version:       1,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)
	result := eval.Evaluate("proj", "prod", "dark-mode", models.EvalContext{Key: "user-1"})

	if result.Reason != models.ReasonOff {
		t.Errorf("expected OFF, got %s", result.Reason)
	}
	if string(result.Value) != "false" {
		t.Errorf("expected false, got %s", result.Value)
	}
}

func TestEvaluate_KillSwitch(t *testing.T) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:feature": {
				ID:         "flag-2",
				ProjectID:  "proj",
				Key:        "feature",
				Type:       models.FlagTypeBoolean,
				Enabled:    true,
				KillSwitch: true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"flag-2:prod": {
				FlagID:        "flag-2",
				EnvironmentID: "prod",
				Enabled:       true,
				OffVariation:  json.RawMessage(`false`),
				Version:       3,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)
	result := eval.Evaluate("proj", "prod", "feature", models.EvalContext{Key: "user-1"})

	if result.Reason != models.ReasonKillSwitch {
		t.Errorf("expected KILL_SWITCH, got %s", result.Reason)
	}
}

func TestEvaluate_RuleMatch(t *testing.T) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:premium": {
				ID:        "flag-3",
				ProjectID: "proj",
				Key:       "premium",
				Type:      models.FlagTypeBoolean,
				Enabled:   true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"flag-3:prod": {
				FlagID:        "flag-3",
				EnvironmentID: "prod",
				Enabled:       true,
				OffVariation:  json.RawMessage(`false`),
				Rules: []models.TargetingRule{
					{
						ID: "rule-1",
						Clauses: []models.Clause{
							{
								Attribute: "plan",
								Operator:  models.OpEquals,
								Values:    []string{"premium"},
							},
						},
						Variation: json.RawMessage(`true`),
						Priority:  1,
					},
				},
				Fallthrough: models.RolloutConfig{
					Variations: []models.WeightedVariation{
						{Variation: json.RawMessage(`false`), Weight: 10000},
					},
				},
				Version: 2,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)

	result := eval.Evaluate("proj", "prod", "premium", models.EvalContext{
		Key:        "user-1",
		Attributes: map[string]any{"plan": "premium"},
	})

	if result.Reason != models.ReasonRuleMatch {
		t.Errorf("expected RULE_MATCH, got %s", result.Reason)
	}
	if string(result.Value) != "true" {
		t.Errorf("expected true, got %s", result.Value)
	}
}

func TestEvaluate_Fallthrough_Rollout(t *testing.T) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:ab-test": {
				ID:        "flag-4",
				ProjectID: "proj",
				Key:       "ab-test",
				Type:      models.FlagTypeString,
				Enabled:   true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"flag-4:prod": {
				FlagID:        "flag-4",
				EnvironmentID: "prod",
				Enabled:       true,
				OffVariation:  json.RawMessage(`"control"`),
				Fallthrough: models.RolloutConfig{
					Variations: []models.WeightedVariation{
						{Variation: json.RawMessage(`"control"`), Weight: 5000},
						{Variation: json.RawMessage(`"variant"`), Weight: 5000},
					},
					BucketBy: "key",
					Seed:     42,
				},
				Version: 1,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)

	controlCount := 0
	variantCount := 0
	total := 10000

	for i := 0; i < total; i++ {
		result := eval.Evaluate("proj", "prod", "ab-test", models.EvalContext{
			Key: fmt.Sprintf("user-%d", i),
		})
		switch string(result.Value) {
		case `"control"`:
			controlCount++
		case `"variant"`:
			variantCount++
		}
	}

	controlPct := float64(controlCount) / float64(total) * 100
	variantPct := float64(variantCount) / float64(total) * 100

	if controlPct < 45 || controlPct > 55 {
		t.Errorf("expected ~50%% control, got %.1f%%", controlPct)
	}
	if variantPct < 45 || variantPct > 55 {
		t.Errorf("expected ~50%% variant, got %.1f%%", variantPct)
	}
}

func TestHashBucket_Deterministic(t *testing.T) {
	b1 := hashBucket("user-123", 42)
	b2 := hashBucket("user-123", 42)
	if b1 != b2 {
		t.Errorf("hash not deterministic: %d != %d", b1, b2)
	}

	b3 := hashBucket("user-456", 42)
	if b1 == b3 {
		t.Error("different users should (usually) get different buckets")
	}
}

func TestHashBucket_Distribution(t *testing.T) {
	buckets := make([]int, 10)
	total := 100000

	for i := 0; i < total; i++ {
		b := hashBucket(fmt.Sprintf("user-%d", i), 0)
		buckets[b/1000]++
	}

	for i, count := range buckets {
		pct := float64(count) / float64(total) * 100
		if pct < 8 || pct > 12 {
			t.Errorf("bucket %d has %.1f%% (expected ~10%%)", i, pct)
		}
	}
}

func BenchmarkEvaluate_SimpleFlag(b *testing.B) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:flag": {
				ID: "f1", ProjectID: "proj", Key: "flag",
				Type: models.FlagTypeBoolean, Enabled: true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"f1:prod": {
				FlagID: "f1", EnvironmentID: "prod", Enabled: true,
				OffVariation: json.RawMessage(`false`),
				Fallthrough: models.RolloutConfig{
					Variations: []models.WeightedVariation{
						{Variation: json.RawMessage(`true`), Weight: 10000},
					},
				},
				Version: 1,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)
	ctx := models.EvalContext{Key: "user-1"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eval.Evaluate("proj", "prod", "flag", ctx)
	}
}

func BenchmarkEvaluate_WithRules(b *testing.B) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:flag": {
				ID: "f1", ProjectID: "proj", Key: "flag",
				Type: models.FlagTypeBoolean, Enabled: true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"f1:prod": {
				FlagID: "f1", EnvironmentID: "prod", Enabled: true,
				OffVariation: json.RawMessage(`false`),
				Rules: []models.TargetingRule{
					{
						Clauses: []models.Clause{
							{Attribute: "country", Operator: models.OpIn, Values: []string{"US", "CA", "GB"}},
							{Attribute: "plan", Operator: models.OpEquals, Values: []string{"premium"}},
						},
						Variation: json.RawMessage(`true`),
					},
				},
				Fallthrough: models.RolloutConfig{
					Variations: []models.WeightedVariation{
						{Variation: json.RawMessage(`false`), Weight: 10000},
					},
				},
				Version: 1,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)
	ctx := models.EvalContext{
		Key:        "user-1",
		Attributes: map[string]any{"country": "US", "plan": "premium"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eval.Evaluate("proj", "prod", "flag", ctx)
	}
}

func BenchmarkEvaluate_RolloutBucketing(b *testing.B) {
	store := &mockStore{
		flags: map[string]*models.Flag{
			"proj:flag": {
				ID: "f1", ProjectID: "proj", Key: "flag",
				Type: models.FlagTypeString, Enabled: true,
			},
		},
		flagEnvs: map[string]*models.FlagEnvironment{
			"f1:prod": {
				FlagID: "f1", EnvironmentID: "prod", Enabled: true,
				OffVariation: json.RawMessage(`"off"`),
				Fallthrough: models.RolloutConfig{
					Variations: []models.WeightedVariation{
						{Variation: json.RawMessage(`"control"`), Weight: 5000},
						{Variation: json.RawMessage(`"variant_a"`), Weight: 3000},
						{Variation: json.RawMessage(`"variant_b"`), Weight: 2000},
					},
					BucketBy: "key",
					Seed:     42,
				},
				Version: 1,
			},
		},
	}

	eval := NewEvaluator(store, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eval.Evaluate("proj", "prod", "flag", models.EvalContext{
			Key: fmt.Sprintf("user-%d", i),
		})
	}
}
