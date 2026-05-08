package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Sriniketh24/rollout/internal/config"
	"github.com/Sriniketh24/rollout/internal/flag"
	"github.com/Sriniketh24/rollout/internal/health"
	"github.com/Sriniketh24/rollout/internal/models"
)

const relayVersion = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	upstreamURL := envOr("UPSTREAM_URL", "http://localhost:8080")
	relayPort := envIntOr("RELAY_PORT", 8081)
	syncInterval := envDurationOr("SYNC_INTERVAL", 10*time.Second)

	h := health.New("rollout-relay", relayVersion)

	store := NewRelayStore()
	evaluator := flag.NewEvaluator(store, 5*time.Minute)

	relay := &Relay{
		upstream:     upstreamURL,
		store:        store,
		evaluator:    evaluator,
		syncInterval: syncInterval,
		logger:       logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go relay.startSync(ctx)

	h.Register("upstream", func(ctx context.Context) error {
		resp, err := relay.client.Get(relay.upstream + "/health/live")
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("upstream unhealthy: %d", resp.StatusCode)
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Handler())
	mux.HandleFunc("GET /health/live", h.LivenessHandler())
	mux.HandleFunc("POST /api/v1/evaluate", relay.handleEvaluate())
	mux.HandleFunc("POST /api/v1/evaluate/{flagKey}", relay.handleEvaluateFlag())
	mux.HandleFunc("GET /api/v1/relay/stats", relay.handleStats())

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, relayPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("relay starting", "addr", addr, "upstream", upstreamURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("relay failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
	logger.Info("relay stopped")
}

type Relay struct {
	upstream     string
	store        *RelayStore
	evaluator    *flag.Evaluator
	syncInterval time.Duration
	logger       *slog.Logger
	client       *http.Client
	lastSync     time.Time
	mu           sync.RWMutex
}

func (rl *Relay) startSync(ctx context.Context) {
	rl.syncOnce(ctx)
	ticker := time.NewTicker(rl.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.syncOnce(ctx)
		}
	}
}

func (rl *Relay) syncOnce(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, "GET", rl.upstream+"/api/v1/relay/rules", nil)
	if err != nil {
		rl.logger.Error("sync request creation failed", "error", err)
		return
	}

	resp, err := rl.client.Do(req)
	if err != nil {
		rl.logger.Error("sync failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rl.logger.Error("sync failed", "status", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		rl.logger.Error("sync read failed", "error", err)
		return
	}

	var rules RulePayload
	if err := json.Unmarshal(body, &rules); err != nil {
		rl.logger.Error("sync unmarshal failed", "error", err)
		return
	}

	rl.store.Update(rules)
	rl.mu.Lock()
	rl.lastSync = time.Now()
	rl.mu.Unlock()
	rl.logger.Info("synced rules", "flags", len(rules.Flags), "environments", len(rules.FlagEnvironments))
}

func (rl *Relay) handleEvaluate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ProjectID     string            `json:"project_id"`
			EnvironmentID string            `json:"environment_id"`
			Context       models.EvalContext `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		results := rl.evaluator.EvaluateAll(req.ProjectID, req.EnvironmentID, req.Context)
		writeJSON(w, http.StatusOK, map[string]any{"evaluations": results})
	}
}

func (rl *Relay) handleEvaluateFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagKey := r.PathValue("flagKey")
		var req struct {
			ProjectID     string            `json:"project_id"`
			EnvironmentID string            `json:"environment_id"`
			Context       models.EvalContext `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		result := rl.evaluator.Evaluate(req.ProjectID, req.EnvironmentID, flagKey, req.Context)
		writeJSON(w, http.StatusOK, result)
	}
}

func (rl *Relay) handleStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rl.mu.RLock()
		lastSync := rl.lastSync
		rl.mu.RUnlock()

		stats := rl.store.Stats()
		stats["last_sync"] = lastSync.Format(time.RFC3339)
		writeJSON(w, http.StatusOK, stats)
	}
}

// RelayStore is an in-memory flag store for the edge relay.
type RelayStore struct {
	mu              sync.RWMutex
	flags           map[string]*models.Flag             // projectID:flagKey -> Flag
	flagEnvironments map[string]*models.FlagEnvironment  // flagID:envID -> FlagEnvironment
	allFlags        map[string][]models.Flag             // projectID:envID -> []Flag
	allFlagEnvs     map[string][]models.FlagEnvironment  // projectID:envID -> []FlagEnvironment
}

type RulePayload struct {
	Flags            []models.Flag            `json:"flags"`
	FlagEnvironments []models.FlagEnvironment `json:"flag_environments"`
}

func NewRelayStore() *RelayStore {
	return &RelayStore{
		flags:            make(map[string]*models.Flag),
		flagEnvironments: make(map[string]*models.FlagEnvironment),
		allFlags:         make(map[string][]models.Flag),
		allFlagEnvs:      make(map[string][]models.FlagEnvironment),
	}
}

func (s *RelayStore) Update(payload RulePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flags = make(map[string]*models.Flag)
	s.flagEnvironments = make(map[string]*models.FlagEnvironment)
	s.allFlags = make(map[string][]models.Flag)
	s.allFlagEnvs = make(map[string][]models.FlagEnvironment)

	for i := range payload.Flags {
		f := &payload.Flags[i]
		key := f.ProjectID + ":" + f.Key
		s.flags[key] = f
	}

	for i := range payload.FlagEnvironments {
		fe := &payload.FlagEnvironments[i]
		key := fe.FlagID + ":" + fe.EnvironmentID
		s.flagEnvironments[key] = fe
	}
}

func (s *RelayStore) GetFlag(projectID, flagKey string) (*models.Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flags[projectID+":"+flagKey]
	if !ok {
		return nil, fmt.Errorf("flag not found: %s", flagKey)
	}
	return f, nil
}

func (s *RelayStore) GetFlagEnvironment(flagID, envID string) (*models.FlagEnvironment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fe, ok := s.flagEnvironments[flagID+":"+envID]
	if !ok {
		return nil, fmt.Errorf("flag environment not found: %s:%s", flagID, envID)
	}
	return fe, nil
}

func (s *RelayStore) GetAllFlagsForEnvironment(projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var flags []models.Flag
	var envs []models.FlagEnvironment

	for _, f := range s.flags {
		if f.ProjectID == projectID {
			flags = append(flags, *f)
			key := f.ID + ":" + envID
			if fe, ok := s.flagEnvironments[key]; ok {
				envs = append(envs, *fe)
			}
		}
	}

	return flags, envs, nil
}

func (s *RelayStore) Stats() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"cached_flags":        len(s.flags),
		"cached_environments": len(s.flagEnvironments),
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
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
