package experiment

import (
	"math"
	"math/rand"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
)

const (
	numSamples    = 50000
	alphaPrior    = 1.0 // Beta(1,1) uniform prior
	betaPrior     = 1.0
)

type VariationData struct {
	Key         string
	Exposures   int64
	Conversions int64
	TotalValue  float64
	IsControl   bool
}

type BayesianEngine struct {
	rng *rand.Rand
}

func NewBayesianEngine() *BayesianEngine {
	return &BayesianEngine{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *BayesianEngine) Analyze(variations []VariationData, guardrails []GuardrailData) models.ExperimentResults {
	variationResults := make([]models.VariationResult, len(variations))

	posteriorSamples := make([][]float64, len(variations))
	for i, v := range variations {
		alpha := alphaPrior + float64(v.Conversions)
		beta := betaPrior + float64(v.Exposures-v.Conversions)
		posteriorSamples[i] = e.sampleBeta(alpha, beta, numSamples)
	}

	controlIdx := 0
	for i, v := range variations {
		if v.IsControl {
			controlIdx = i
			break
		}
	}

	for i, v := range variations {
		cr := float64(0)
		if v.Exposures > 0 {
			cr = float64(v.Conversions) / float64(v.Exposures)
		}

		ptb := e.probabilityToBeatControl(posteriorSamples, i, controlIdx)
		loss := e.expectedLoss(posteriorSamples, i, controlIdx)
		ci := e.credibleInterval(posteriorSamples[i], 0.95)

		var guardrailResults []models.GuardrailResult
		for _, g := range guardrails {
			if gr, ok := g.VariationResults[v.Key]; ok {
				guardrailResults = append(guardrailResults, gr)
			}
		}

		variationResults[i] = models.VariationResult{
			VariationKey:      v.Key,
			Exposures:         v.Exposures,
			Conversions:       v.Conversions,
			ConversionRate:    cr,
			ProbabilityToBeat: ptb,
			ExpectedLoss:      loss,
			CredibleInterval:  ci,
			IsControl:         v.IsControl,
			GuardrailResults:  guardrailResults,
		}
	}

	totalExposures := int64(0)
	for _, v := range variations {
		totalExposures += v.Exposures
	}

	return models.ExperimentResults{
		ComputedAt:       time.Now().UTC(),
		TotalExposures:   totalExposures,
		VariationResults: variationResults,
		Recommendation:   e.generateRecommendation(variationResults),
	}
}

func (e *BayesianEngine) probabilityToBeatControl(samples [][]float64, varIdx, controlIdx int) float64 {
	if varIdx == controlIdx {
		return 0
	}
	wins := 0
	for j := 0; j < numSamples; j++ {
		if samples[varIdx][j] > samples[controlIdx][j] {
			wins++
		}
	}
	return float64(wins) / float64(numSamples)
}

func (e *BayesianEngine) expectedLoss(samples [][]float64, varIdx, controlIdx int) float64 {
	if varIdx == controlIdx {
		return 0
	}
	totalLoss := 0.0
	for j := 0; j < numSamples; j++ {
		loss := samples[controlIdx][j] - samples[varIdx][j]
		if loss > 0 {
			totalLoss += loss
		}
	}
	return totalLoss / float64(numSamples)
}

func (e *BayesianEngine) credibleInterval(samples []float64, level float64) [2]float64 {
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sortFloat64s(sorted)

	lower := (1 - level) / 2
	upper := 1 - lower
	lowerIdx := int(math.Floor(lower * float64(len(sorted))))
	upperIdx := int(math.Floor(upper * float64(len(sorted))))

	if upperIdx >= len(sorted) {
		upperIdx = len(sorted) - 1
	}

	return [2]float64{sorted[lowerIdx], sorted[upperIdx]}
}

func (e *BayesianEngine) sampleBeta(alpha, beta float64, n int) []float64 {
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		samples[i] = e.betaSample(alpha, beta)
	}
	return samples
}

// betaSample generates a sample from Beta(alpha, beta) using the gamma distribution method.
func (e *BayesianEngine) betaSample(alpha, beta float64) float64 {
	x := e.gammaSample(alpha)
	y := e.gammaSample(beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// gammaSample generates a sample from Gamma(shape, 1) using Marsaglia and Tsang's method.
func (e *BayesianEngine) gammaSample(shape float64) float64 {
	if shape < 1 {
		return e.gammaSample(shape+1) * math.Pow(e.rng.Float64(), 1.0/shape)
	}

	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)

	for {
		var x, v float64
		for {
			x = e.rng.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := e.rng.Float64()
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

func (e *BayesianEngine) generateRecommendation(results []models.VariationResult) string {
	if len(results) < 2 {
		return "Insufficient variations for comparison."
	}

	var bestNonControl *models.VariationResult
	for i := range results {
		if !results[i].IsControl {
			if bestNonControl == nil || results[i].ProbabilityToBeat > bestNonControl.ProbabilityToBeat {
				bestNonControl = &results[i]
			}
		}
	}

	if bestNonControl == nil {
		return "No treatment variations found."
	}

	guardrailDegraded := false
	for _, gr := range bestNonControl.GuardrailResults {
		if gr.Degraded {
			guardrailDegraded = true
			break
		}
	}

	switch {
	case bestNonControl.ProbabilityToBeat >= 0.95 && !guardrailDegraded:
		return "Ship: " + bestNonControl.VariationKey + " beats control with " +
			formatPercent(bestNonControl.ProbabilityToBeat) + " probability. All guardrails pass."
	case bestNonControl.ProbabilityToBeat >= 0.95 && guardrailDegraded:
		return "Investigate: " + bestNonControl.VariationKey + " beats control with " +
			formatPercent(bestNonControl.ProbabilityToBeat) + " probability, but guardrail metrics degraded."
	case bestNonControl.ProbabilityToBeat >= 0.80:
		return "Continue: " + bestNonControl.VariationKey + " trending positive (" +
			formatPercent(bestNonControl.ProbabilityToBeat) + "), but more data needed for confidence."
	default:
		return "No clear winner yet. Continue collecting data."
	}
}

func formatPercent(f float64) string {
	pct := f * 100
	whole := int(pct)
	frac := int((pct - float64(whole)) * 10)
	return intToStr(whole) + "." + intToStr(frac) + "%"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func sortFloat64s(a []float64) {
	quicksort(a, 0, len(a)-1)
}

func quicksort(a []float64, lo, hi int) {
	if lo < hi {
		p := partition(a, lo, hi)
		quicksort(a, lo, p-1)
		quicksort(a, p+1, hi)
	}
}

func partition(a []float64, lo, hi int) int {
	pivot := a[hi]
	i := lo
	for j := lo; j < hi; j++ {
		if a[j] < pivot {
			a[i], a[j] = a[j], a[i]
			i++
		}
	}
	a[i], a[hi] = a[hi], a[i]
	return i
}

type GuardrailData struct {
	MetricKey        string
	VariationResults map[string]models.GuardrailResult
}
