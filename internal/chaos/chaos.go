package chaos

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/Sriniketh24/rollout/internal/config"
	"github.com/Sriniketh24/rollout/internal/models"
)

type Engine struct {
	mu      sync.RWMutex
	cfg     config.ChaosConfig
	rng     *rand.Rand
	enabled bool
	stats   Stats
}

type Stats struct {
	InjectedFailures    int64 `json:"injected_failures"`
	InjectedLatency     int64 `json:"injected_latency"`
	StaleCacheResponses int64 `json:"stale_cache_responses"`
	TotalRequests       int64 `json:"total_requests"`
}

func NewEngine(cfg config.ChaosConfig) *Engine {
	return &Engine{
		cfg:     cfg,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		enabled: cfg.Enabled,
	}
}

func (e *Engine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

func (e *Engine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

func (e *Engine) UpdateConfig(cfg config.ChaosConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
	e.enabled = cfg.Enabled
}

func (e *Engine) GetStats() Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

func (e *Engine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = Stats{}
}

func (e *Engine) MaybeInjectLatency(ctx context.Context) {
	e.mu.Lock()
	e.stats.TotalRequests++
	if !e.enabled || e.cfg.LatencyInjection == 0 {
		e.mu.Unlock()
		return
	}
	if e.rng.Float64() < e.cfg.FailureRate {
		delay := e.cfg.LatencyInjection
		e.stats.InjectedLatency++
		e.mu.Unlock()
		select {
		case <-time.After(delay):
		case <-ctx.Done():
		}
		return
	}
	e.mu.Unlock()
}

func (e *Engine) ShouldFail() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.enabled || e.cfg.FailureRate == 0 {
		return false
	}
	if e.rng.Float64() < e.cfg.FailureRate {
		e.stats.InjectedFailures++
		return true
	}
	return false
}

func (e *Engine) ShouldReturnStaleCache() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.enabled || e.cfg.StaleCacheRate == 0 {
		return false
	}
	if e.rng.Float64() < e.cfg.StaleCacheRate {
		e.stats.StaleCacheResponses++
		return true
	}
	return false
}

func (e *Engine) MaybeCorruptEvaluation(result models.EvalResult) models.EvalResult {
	if !e.ShouldFail() {
		return result
	}
	corrupted := result
	corrupted.Value = json.RawMessage(`null`)
	corrupted.Reason = models.ReasonError
	return corrupted
}
