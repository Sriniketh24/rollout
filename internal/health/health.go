package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Checker func(ctx context.Context) error

type Health struct {
	mu       sync.RWMutex
	checks   map[string]Checker
	version  string
	service  string
	startedAt time.Time
}

func New(service, version string) *Health {
	return &Health{
		checks:    make(map[string]Checker),
		version:   version,
		service:   service,
		startedAt: time.Now(),
	}
}

func (h *Health) Register(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = checker
}

func (h *Health) Check(ctx context.Context) (status string, checks map[string]string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	checks = make(map[string]string, len(h.checks))
	status = "healthy"

	for name, checker := range h.checks {
		if err := checker(ctx); err != nil {
			checks[name] = "unhealthy: " + err.Error()
			status = "degraded"
		} else {
			checks[name] = "healthy"
		}
	}
	return
}

func (h *Health) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status, checks := h.Check(ctx)

		resp := map[string]any{
			"service": h.service,
			"status":  status,
			"version": h.version,
			"uptime":  time.Since(h.startedAt).String(),
			"checks":  checks,
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func (h *Health) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}
