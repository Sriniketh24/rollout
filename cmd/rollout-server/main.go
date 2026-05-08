package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sriniketh24/rollout/internal/auth"
	"github.com/Sriniketh24/rollout/internal/chaos"
	"github.com/Sriniketh24/rollout/internal/config"
	"github.com/Sriniketh24/rollout/internal/health"
	"github.com/Sriniketh24/rollout/internal/middleware"
	"github.com/Sriniketh24/rollout/internal/streaming"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	h := health.New("rollout-server", version)
	chaosEngine := chaos.NewEngine(cfg.Chaos)
	sseHub := streaming.NewHub()

	_ = chaosEngine
	_ = sseHub

	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /health", h.Handler())
	mux.HandleFunc("GET /health/live", h.LivenessHandler())

	// API v1 routes
	mux.HandleFunc("GET /api/v1/projects", handleListProjects())
	mux.HandleFunc("POST /api/v1/projects", handleCreateProject())
	mux.HandleFunc("GET /api/v1/projects/{projectID}", handleGetProject())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments", handleListEnvironments())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments", handleCreateEnvironment())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/flags", handleListFlags())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags", handleCreateFlag())
	mux.HandleFunc("GET /api/v1/projects/{projectID}/flags/{flagKey}", handleGetFlag())
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}", handleUpdateFlag())
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/flags/{flagKey}", handleDeleteFlag())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/toggle", handleToggleFlag())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/kill", handleKillSwitch())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/rollback", handleRollback())

	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}/environments/{envID}", handleUpdateFlagEnvironment())
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}/environments/{envID}/rules", handleUpdateTargetingRules())

	mux.HandleFunc("POST /api/v1/projects/{projectID}/evaluate", handleEvaluateFlags())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/evaluate/{flagKey}", handleEvaluateFlag())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments", handleListExperiments())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments", handleCreateExperiment())
	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments/{experimentID}", handleGetExperiment())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments/{experimentID}/start", handleStartExperiment())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments/{experimentID}/stop", handleStopExperiment())
	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments/{experimentID}/results", handleExperimentResults())

	mux.HandleFunc("POST /api/v1/events/exposures", handleIngestExposures())
	mux.HandleFunc("POST /api/v1/events/conversions", handleIngestConversions())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/audit", handleListAuditLogs())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/stream/{envID}", handleSSEStream(sseHub))

	mux.HandleFunc("POST /api/v1/auth/login", handleLogin())
	mux.HandleFunc("POST /api/v1/auth/register", handleRegister())
	mux.HandleFunc("GET /api/v1/auth/me", handleMe())

	mux.HandleFunc("GET /api/v1/projects/{projectID}/users", handleListUsers())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/users", handleInviteUser())
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/users/{userID}/role", handleUpdateUserRole())

	mux.HandleFunc("POST /api/v1/projects/{projectID}/sync", handleGitOpsSync())
	mux.HandleFunc("POST /api/v1/projects/{projectID}/diff", handleGitOpsDiff())

	mux.HandleFunc("GET /api/v1/chaos/stats", handleChaosStats(chaosEngine))
	mux.HandleFunc("POST /api/v1/chaos/toggle", handleChaosToggle(chaosEngine))
	mux.HandleFunc("PUT /api/v1/chaos/config", handleChaosConfig(chaosEngine))

	handler := middleware.Chain(
		mux,
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.CORS([]string{"*"}),
		middleware.RateLimit(100),
	)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("server starting", "addr", addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "error", err)
	}
	logger.Info("server stopped")
}

// --- Handler stubs (each returns a functional handler) ---

func handleListProjects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"projects": []any{}, "total": 0})
	}
}

func handleCreateProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": "generated", "name": req.Name, "key": req.Key})
	}
}

func handleGetProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("projectID")
		writeJSON(w, http.StatusOK, map[string]string{"id": projectID})
	}
}

func handleListEnvironments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"environments": []any{}})
	}
}

func handleCreateEnvironment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Key   string `json:"key"`
			Color string `json:"color"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "key": req.Key})
	}
}

func handleListFlags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"flags": []any{}, "total": 0})
	}
}

func handleCreateFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusCreated, req)
	}
}

func handleGetFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagKey := r.PathValue("flagKey")
		writeJSON(w, http.StatusOK, map[string]string{"key": flagKey})
	}
}

func handleUpdateFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, req)
	}
}

func handleDeleteFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"deleted": "true"})
	}
}

func handleToggleFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"toggled": "true"})
	}
}

func handleKillSwitch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"kill_switch": "activated"})
	}
}

func handleRollback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"rolled_back": "true"})
	}
}

func handleUpdateFlagEnvironment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, req)
	}
}

func handleUpdateTargetingRules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, req)
	}
}

func handleEvaluateFlags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"evaluations": []any{}})
	}
}

func handleEvaluateFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagKey := r.PathValue("flagKey")
		writeJSON(w, http.StatusOK, map[string]any{"flag_key": flagKey, "value": true, "reason": "FALLTHROUGH"})
	}
}

func handleListExperiments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"experiments": []any{}})
	}
}

func handleCreateExperiment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusCreated, req)
	}
}

func handleGetExperiment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("experimentID")})
	}
}

func handleStartExperiment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
	}
}

func handleStopExperiment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
	}
}

func handleExperimentResults() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"results": nil})
	}
}

func handleIngestExposures() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Events []json.RawMessage `json:"events"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusAccepted, map[string]int{"accepted": len(req.Events)})
	}
}

func handleIngestConversions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Events []json.RawMessage `json:"events"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusAccepted, map[string]int{"accepted": len(req.Events)})
	}
}

func handleListAuditLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []any{}, "total": 0})
	}
}

func handleSSEStream(hub *streaming.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("projectID")
		envID := r.PathValue("envID")
		hub.Subscribe(projectID, envID, w, r)
	}
}

func handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"token": "jwt-token"})
	}
}

func handleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"id": "new-user"})
	}
}

func handleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUser(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}

func handleListUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"users": []any{}})
	}
}

func handleInviteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"invited": "true"})
	}
}

func handleUpdateUserRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"updated": "true"})
	}
}

func handleGitOpsSync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"synced": true, "changes": 0})
	}
}

func handleGitOpsDiff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"diffs": []any{}})
	}
}

func handleChaosStats(engine *chaos.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, engine.GetStats())
	}
}

func handleChaosToggle(engine *chaos.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		engine.SetEnabled(req.Enabled)
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
	}
}

func handleChaosConfig(engine *chaos.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req config.ChaosConfig
		json.NewDecoder(r.Body).Decode(&req)
		engine.UpdateConfig(req)
		writeJSON(w, http.StatusOK, map[string]string{"updated": "true"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
