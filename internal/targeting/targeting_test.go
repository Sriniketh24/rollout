package targeting

import (
	"encoding/json"
	"testing"

	"github.com/Sriniketh24/rollout/internal/models"
)

func TestEvaluateClause_Equals(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for country eq US")
	}
}

func TestEvaluateClause_NotEquals(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "plan", Operator: models.OpNotEquals, Values: []string{"free"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"plan": "premium"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for plan neq free when plan=premium")
	}
}

func TestEvaluateClause_In(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpIn, Values: []string{"US", "CA", "GB"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "CA"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for country in [US,CA,GB] when country=CA")
	}
}

func TestEvaluateClause_NotIn(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpNotIn, Values: []string{"CN", "RU"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for country not_in [CN,RU] when country=US")
	}
}

func TestEvaluateClause_Contains(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "email", Operator: models.OpContains, Values: []string{"@company.com"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"email": "alice@company.com"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for email contains @company.com")
	}
}

func TestEvaluateClause_StartsWith(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "key", Operator: models.OpStartsWith, Values: []string{"user-"}}
	ctx := models.EvalContext{Key: "user-123", Attributes: map[string]any{"key": "user-abc"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for key starts_with user-")
	}
}

func TestEvaluateClause_EndsWith(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "email", Operator: models.OpEndsWith, Values: []string{".edu"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"email": "student@mit.edu"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for email ends_with .edu")
	}
}

func TestEvaluateClause_Regex(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "version", Operator: models.OpRegex, Values: []string{`^v[0-9]+`}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"version": "v2.3.1"}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for version matching ^v[0-9]+")
	}
}

func TestEvaluateClause_Negate(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}, Negate: true}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US"}}
	if e.EvaluateClause(clause, ctx) {
		t.Fatal("expected false for negated eq match")
	}
}

func TestEvaluateClause_MissingAttribute(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{}}
	if e.EvaluateClause(clause, ctx) {
		t.Fatal("expected false when attribute is missing from context")
	}
}

func TestEvaluateClause_KeyAttribute(t *testing.T) {
	e := New()
	clause := models.Clause{Attribute: "key", Operator: models.OpEquals, Values: []string{"user-42"}}
	ctx := models.EvalContext{Key: "user-42", Attributes: map[string]any{}}
	if !e.EvaluateClause(clause, ctx) {
		t.Fatal("expected true for attribute=key mapping to ctx.Key")
	}
}

func TestEvaluateRules_FirstMatchWins(t *testing.T) {
	e := New()
	rules := []models.TargetingRule{
		{
			ID:        "rule-1",
			Clauses:   []models.Clause{{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}}},
			Variation: json.RawMessage(`"on"`),
			Priority:  1,
		},
		{
			ID:        "rule-2",
			Clauses:   []models.Clause{{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}}},
			Variation: json.RawMessage(`"off"`),
			Priority:  2,
		},
	}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US"}}
	matched, ok := e.EvaluateRules(rules, ctx)
	if !ok {
		t.Fatal("expected a match")
	}
	if matched.ID != "rule-1" {
		t.Fatalf("expected rule-1 to match first, got %s", matched.ID)
	}
}

func TestEvaluateRules_NoMatch(t *testing.T) {
	e := New()
	rules := []models.TargetingRule{
		{
			ID:      "rule-1",
			Clauses: []models.Clause{{Attribute: "country", Operator: models.OpEquals, Values: []string{"GB"}}},
		},
	}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US"}}
	_, ok := e.EvaluateRules(rules, ctx)
	if ok {
		t.Fatal("expected no match")
	}
}

func TestEvaluateRules_MultipleClausesAND(t *testing.T) {
	e := New()
	rules := []models.TargetingRule{
		{
			ID: "rule-1",
			Clauses: []models.Clause{
				{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}},
				{Attribute: "plan", Operator: models.OpEquals, Values: []string{"premium"}},
			},
			Variation: json.RawMessage(`"on"`),
		},
	}

	// Both match
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US", "plan": "premium"}}
	matched, ok := e.EvaluateRules(rules, ctx)
	if !ok || matched.ID != "rule-1" {
		t.Fatal("expected match when both clauses satisfied")
	}

	// Only one matches
	ctx2 := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US", "plan": "free"}}
	_, ok2 := e.EvaluateRules(rules, ctx2)
	if ok2 {
		t.Fatal("expected no match when only one clause satisfied")
	}
}

func TestMatchesSegment_Excluded(t *testing.T) {
	e := New()
	segment := &Segment{
		ID:       "seg-1",
		Excluded: []string{"user-bad"},
		Included: []string{"user-bad"}, // included too, but excluded takes precedence
		Rules:    []models.Clause{},
	}
	ctx := models.EvalContext{Key: "user-bad", Attributes: map[string]any{}}
	if e.MatchesSegment(segment, ctx) {
		t.Fatal("expected false for excluded user")
	}
}

func TestMatchesSegment_Included(t *testing.T) {
	e := New()
	segment := &Segment{
		ID:       "seg-1",
		Included: []string{"user-vip"},
		Rules: []models.Clause{
			// Rule that would NOT match
			{Attribute: "plan", Operator: models.OpEquals, Values: []string{"enterprise"}},
		},
	}
	ctx := models.EvalContext{Key: "user-vip", Attributes: map[string]any{"plan": "free"}}
	if !e.MatchesSegment(segment, ctx) {
		t.Fatal("expected true for included user even if rules wouldn't match")
	}
}

func TestMatchesSegment_RuleBased(t *testing.T) {
	e := New()
	segment := &Segment{
		ID: "seg-1",
		Rules: []models.Clause{
			{Attribute: "country", Operator: models.OpEquals, Values: []string{"US"}},
			{Attribute: "plan", Operator: models.OpEquals, Values: []string{"premium"}},
		},
	}

	// All rules match
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US", "plan": "premium"}}
	if !e.MatchesSegment(segment, ctx) {
		t.Fatal("expected true when all segment rules match")
	}

	// One rule fails
	ctx2 := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "US", "plan": "free"}}
	if e.MatchesSegment(segment, ctx2) {
		t.Fatal("expected false when a segment rule fails")
	}
}

func BenchmarkEvaluateClause(b *testing.B) {
	e := New()
	clause := models.Clause{Attribute: "country", Operator: models.OpIn, Values: []string{"US", "CA", "GB", "AU", "NZ"}}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{"country": "AU"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.EvaluateClause(clause, ctx)
	}
}

func BenchmarkEvaluateRules(b *testing.B) {
	e := New()
	rules := []models.TargetingRule{
		{
			ID: "rule-1",
			Clauses: []models.Clause{
				{Attribute: "country", Operator: models.OpEquals, Values: []string{"GB"}},
			},
		},
		{
			ID: "rule-2",
			Clauses: []models.Clause{
				{Attribute: "country", Operator: models.OpIn, Values: []string{"US", "CA"}},
				{Attribute: "plan", Operator: models.OpEquals, Values: []string{"premium"}},
			},
		},
		{
			ID: "rule-3",
			Clauses: []models.Clause{
				{Attribute: "email", Operator: models.OpRegex, Values: []string{`^.*@company\.com$`}},
			},
		},
	}
	ctx := models.EvalContext{Key: "user-1", Attributes: map[string]any{
		"country": "US",
		"plan":    "premium",
		"email":   "alice@company.com",
	}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.EvaluateRules(rules, ctx)
	}
}
