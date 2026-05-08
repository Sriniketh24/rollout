package flag

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
)

type FlagStore interface {
	GetFlag(projectID, flagKey string) (*models.Flag, error)
	GetFlagEnvironment(flagID, envID string) (*models.FlagEnvironment, error)
	GetAllFlagsForEnvironment(projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error)
}

type Evaluator struct {
	store     FlagStore
	cache     sync.Map
	cacheTTL  time.Duration
}

func NewEvaluator(store FlagStore, cacheTTL time.Duration) *Evaluator {
	return &Evaluator{
		store:    store,
		cacheTTL: cacheTTL,
	}
}

func (e *Evaluator) Evaluate(projectID, envID, flagKey string, ctx models.EvalContext) models.EvalResult {
	now := time.Now()

	flag, flagEnv, err := e.resolveFlag(projectID, envID, flagKey)
	if err != nil {
		return models.EvalResult{
			FlagKey:   flagKey,
			Value:     json.RawMessage(`null`),
			Reason:    models.ReasonError,
			Timestamp: now,
		}
	}

	if flag.KillSwitch {
		return models.EvalResult{
			FlagKey:   flagKey,
			Value:     flagEnv.OffVariation,
			Reason:    models.ReasonKillSwitch,
			Version:   flagEnv.Version,
			Timestamp: now,
		}
	}

	if !flag.Enabled || !flagEnv.Enabled {
		return models.EvalResult{
			FlagKey:   flagKey,
			Value:     flagEnv.OffVariation,
			Reason:    models.ReasonOff,
			Version:   flagEnv.Version,
			Timestamp: now,
		}
	}

	if len(flag.DependsOn) > 0 {
		for _, dep := range flag.DependsOn {
			depResult := e.Evaluate(projectID, envID, dep, ctx)
			if depResult.Reason == models.ReasonOff || depResult.Reason == models.ReasonError {
				return models.EvalResult{
					FlagKey:   flagKey,
					Value:     flagEnv.OffVariation,
					Reason:    models.ReasonDependency,
					Version:   flagEnv.Version,
					Timestamp: now,
				}
			}
		}
	}

	for _, rule := range flagEnv.Rules {
		if matchesRule(rule, ctx) {
			value := resolveVariation(rule, ctx)
			return models.EvalResult{
				FlagKey:   flagKey,
				Value:     value,
				Reason:    models.ReasonRuleMatch,
				Version:   flagEnv.Version,
				Timestamp: now,
			}
		}
	}

	value := resolveFallthrough(flagEnv.Fallthrough, ctx)
	return models.EvalResult{
		FlagKey:   flagKey,
		Value:     value,
		Reason:    models.ReasonFallthrough,
		Version:   flagEnv.Version,
		Timestamp: now,
	}
}

func (e *Evaluator) EvaluateAll(projectID, envID string, ctx models.EvalContext) []models.EvalResult {
	flags, flagEnvs, err := e.store.GetAllFlagsForEnvironment(projectID, envID)
	if err != nil {
		return nil
	}

	envMap := make(map[string]models.FlagEnvironment)
	for _, fe := range flagEnvs {
		envMap[fe.FlagID] = fe
	}

	results := make([]models.EvalResult, 0, len(flags))
	for _, f := range flags {
		result := e.Evaluate(projectID, envID, f.Key, ctx)
		results = append(results, result)
	}
	return results
}

func (e *Evaluator) resolveFlag(projectID, envID, flagKey string) (*models.Flag, *models.FlagEnvironment, error) {
	flag, err := e.store.GetFlag(projectID, flagKey)
	if err != nil {
		return nil, nil, err
	}

	flagEnv, err := e.store.GetFlagEnvironment(flag.ID, envID)
	if err != nil {
		return nil, nil, err
	}

	return flag, flagEnv, nil
}

func (e *Evaluator) InvalidateCache(flagKey string) {
	e.cache.Delete(flagKey)
}

func matchesRule(rule models.TargetingRule, ctx models.EvalContext) bool {
	for _, clause := range rule.Clauses {
		if !matchesClause(clause, ctx) {
			return false
		}
	}
	return true
}

func matchesClause(clause models.Clause, ctx models.EvalContext) bool {
	attrVal := getAttributeValue(clause.Attribute, ctx)
	result := evaluateOperator(clause.Operator, attrVal, clause.Values)
	if clause.Negate {
		return !result
	}
	return result
}

func getAttributeValue(attr string, ctx models.EvalContext) string {
	if attr == "key" {
		return ctx.Key
	}
	if v, ok := ctx.Attributes[attr]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func evaluateOperator(op models.Operator, attrVal string, values []string) bool {
	switch op {
	case models.OpEquals:
		return len(values) > 0 && attrVal == values[0]
	case models.OpNotEquals:
		return len(values) > 0 && attrVal != values[0]
	case models.OpIn:
		for _, v := range values {
			if attrVal == v {
				return true
			}
		}
		return false
	case models.OpNotIn:
		for _, v := range values {
			if attrVal == v {
				return false
			}
		}
		return true
	case models.OpContains:
		if len(values) == 0 {
			return false
		}
		return containsStr(attrVal, values[0])
	case models.OpStartsWith:
		if len(values) == 0 {
			return false
		}
		return len(attrVal) >= len(values[0]) && attrVal[:len(values[0])] == values[0]
	case models.OpEndsWith:
		if len(values) == 0 {
			return false
		}
		return len(attrVal) >= len(values[0]) && attrVal[len(attrVal)-len(values[0]):] == values[0]
	case models.OpGreaterThan:
		return compareNumeric(attrVal, values) > 0
	case models.OpLessThan:
		return compareNumeric(attrVal, values) < 0
	default:
		return false
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func compareNumeric(attrVal string, values []string) int {
	if len(values) == 0 {
		return 0
	}
	var a, b float64
	fmt.Sscanf(attrVal, "%f", &a)
	fmt.Sscanf(values[0], "%f", &b)
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

func resolveVariation(rule models.TargetingRule, ctx models.EvalContext) json.RawMessage {
	if rule.Rollout != nil {
		return rolloutVariation(*rule.Rollout, ctx)
	}
	return rule.Variation
}

func resolveFallthrough(ft models.RolloutConfig, ctx models.EvalContext) json.RawMessage {
	return rolloutVariation(ft, ctx)
}

func rolloutVariation(rc models.RolloutConfig, ctx models.EvalContext) json.RawMessage {
	if len(rc.Variations) == 0 {
		return json.RawMessage(`null`)
	}
	if len(rc.Variations) == 1 {
		return rc.Variations[0].Variation
	}

	bucketKey := ctx.Key
	if rc.BucketBy != "" {
		if v, ok := ctx.Attributes[rc.BucketBy]; ok {
			bucketKey = fmt.Sprintf("%v", v)
		}
	}

	bucket := hashBucket(bucketKey, rc.Seed)

	cumulative := 0
	for _, wv := range rc.Variations {
		cumulative += wv.Weight
		if bucket < cumulative {
			return wv.Variation
		}
	}

	return rc.Variations[len(rc.Variations)-1].Variation
}

// hashBucket produces a deterministic bucket in [0, 10000) for sticky bucketing.
func hashBucket(key string, seed int64) int {
	h := sha256.New()
	seedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBytes, uint64(seed))
	h.Write(seedBytes)
	h.Write([]byte(key))
	sum := h.Sum(nil)
	val := binary.BigEndian.Uint32(sum[:4])
	return int(val % 10000)
}
