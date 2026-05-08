package experiment

import (
	"math"
	"strings"
	"testing"

	"github.com/Sriniketh24/rollout/internal/models"
)

func TestAnalyze_ClearWinner(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 500, IsControl: true},
		{Key: "variant", Exposures: 10000, Conversions: 800, IsControl: false},
	}

	results := engine.Analyze(variations, nil)

	var variant *models.VariationResult
	for i := range results.VariationResults {
		if results.VariationResults[i].VariationKey == "variant" {
			variant = &results.VariationResults[i]
			break
		}
	}

	if variant == nil {
		t.Fatal("variant result not found")
	}

	if variant.ProbabilityToBeat < 0.90 {
		t.Errorf("expected ProbabilityToBeat > 0.90 for clear winner, got %f", variant.ProbabilityToBeat)
	}

	if !strings.Contains(results.Recommendation, "Ship") {
		t.Errorf("expected recommendation to contain 'Ship', got %q", results.Recommendation)
	}
}

func TestAnalyze_NoWinner(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 500, IsControl: true},
		{Key: "variant", Exposures: 10000, Conversions: 510, IsControl: false},
	}

	results := engine.Analyze(variations, nil)

	containsExpected := strings.Contains(results.Recommendation, "No clear winner") ||
		strings.Contains(results.Recommendation, "Continue")

	if !containsExpected {
		t.Errorf("expected recommendation to contain 'No clear winner' or 'Continue', got %q", results.Recommendation)
	}
}

func TestAnalyze_ConversionRateCalculation(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 100, Conversions: 20, IsControl: true},
		{Key: "variant", Exposures: 100, Conversions: 30, IsControl: false},
	}

	results := engine.Analyze(variations, nil)

	var control *models.VariationResult
	for i := range results.VariationResults {
		if results.VariationResults[i].VariationKey == "control" {
			control = &results.VariationResults[i]
			break
		}
	}

	if control == nil {
		t.Fatal("control result not found")
	}

	if math.Abs(control.ConversionRate-0.20) > 1e-9 {
		t.Errorf("expected ConversionRate = 0.20, got %f", control.ConversionRate)
	}
}

func TestAnalyze_CredibleIntervalContainsTrueRate(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 2000, IsControl: true},
		{Key: "variant", Exposures: 10000, Conversions: 2100, IsControl: false},
	}

	results := engine.Analyze(variations, nil)

	var control *models.VariationResult
	for i := range results.VariationResults {
		if results.VariationResults[i].VariationKey == "control" {
			control = &results.VariationResults[i]
			break
		}
	}

	if control == nil {
		t.Fatal("control result not found")
	}

	trueRate := 0.20
	ci := control.CredibleInterval
	if ci[0] > trueRate || ci[1] < trueRate {
		t.Errorf("expected 95%% credible interval [%f, %f] to contain true rate %f", ci[0], ci[1], trueRate)
	}
}

func TestAnalyze_GuardrailDegraded(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 500, IsControl: true},
		{Key: "variant", Exposures: 10000, Conversions: 800, IsControl: false},
	}

	guardrails := []GuardrailData{
		{
			MetricKey: "latency_p99",
			VariationResults: map[string]models.GuardrailResult{
				"variant": {
					MetricKey:      "latency_p99",
					ControlMean:    200.0,
					VariantMean:    350.0,
					RelativeChange: 0.75,
					Degraded:       true,
				},
			},
		},
	}

	results := engine.Analyze(variations, guardrails)

	if !strings.Contains(results.Recommendation, "Investigate") {
		t.Errorf("expected recommendation to contain 'Investigate' when guardrail is degraded, got %q", results.Recommendation)
	}
}

func TestAnalyze_SingleVariation(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 500, IsControl: true},
	}

	results := engine.Analyze(variations, nil)

	if !strings.Contains(results.Recommendation, "Insufficient variations") {
		t.Errorf("expected recommendation to contain 'Insufficient variations', got %q", results.Recommendation)
	}
}

func TestAnalyze_TotalExposures(t *testing.T) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 5000, Conversions: 250, IsControl: true},
		{Key: "variant_a", Exposures: 3000, Conversions: 200, IsControl: false},
		{Key: "variant_b", Exposures: 7000, Conversions: 500, IsControl: false},
	}

	results := engine.Analyze(variations, nil)

	expectedTotal := int64(5000 + 3000 + 7000)
	if results.TotalExposures != expectedTotal {
		t.Errorf("expected TotalExposures = %d, got %d", expectedTotal, results.TotalExposures)
	}
}

func BenchmarkAnalyze(b *testing.B) {
	engine := NewBayesianEngine()

	variations := []VariationData{
		{Key: "control", Exposures: 10000, Conversions: 500, IsControl: true},
		{Key: "variant_a", Exposures: 10000, Conversions: 600, IsControl: false},
		{Key: "variant_b", Exposures: 10000, Conversions: 700, IsControl: false},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Analyze(variations, nil)
	}
}
