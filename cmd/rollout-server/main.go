package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sriniketh24/rollout/internal/auth"
	"github.com/Sriniketh24/rollout/internal/chaos"
	"github.com/Sriniketh24/rollout/internal/config"
	"github.com/Sriniketh24/rollout/internal/flag"
	"github.com/Sriniketh24/rollout/internal/health"
	"github.com/Sriniketh24/rollout/internal/middleware"
	"github.com/Sriniketh24/rollout/internal/models"
	"github.com/Sriniketh24/rollout/internal/store"
	"github.com/Sriniketh24/rollout/internal/streaming"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Connect to Postgres
	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN())
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Warn("postgres not reachable at startup, continuing anyway", "error", err)
	}

	db := store.New(pool)
	evaluator := flag.NewEvaluator(db, 30*time.Second)
	authenticator := auth.New(db, cfg.Auth.JWTSecret, 24*time.Hour)

	h := health.New("rollout-server", version)
	h.Register("postgres", func(_ context.Context) error { return pool.Ping(context.Background()) })

	chaosEngine := chaos.NewEngine(cfg.Chaos)
	sseHub := streaming.NewHub()

	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /health", h.Handler())
	mux.HandleFunc("GET /health/live", h.LivenessHandler())

	// Projects
	mux.HandleFunc("GET /api/v1/projects", handleListProjects(db))
	mux.HandleFunc("POST /api/v1/projects", handleCreateProject(db))
	mux.HandleFunc("GET /api/v1/projects/{projectID}", handleGetProject(db))

	// Environments
	mux.HandleFunc("GET /api/v1/projects/{projectID}/environments", handleListEnvironments(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/environments", handleCreateEnvironment(db))

	// Flags
	mux.HandleFunc("GET /api/v1/projects/{projectID}/flags", handleListFlags(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags", handleCreateFlag(db))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/flags/{flagKey}", handleGetFlag(db))
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}", handleUpdateFlag(db))
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/flags/{flagKey}", handleDeleteFlag(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/toggle", handleToggleFlag(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/kill", handleKillSwitch(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/flags/{flagKey}/rollback", handleRollback(db))

	// Flag-Environment configuration
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}/environments/{envID}", handleUpdateFlagEnvironment(db))
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/flags/{flagKey}/environments/{envID}/rules", handleUpdateTargetingRules(db))

	// Evaluation
	mux.HandleFunc("POST /api/v1/projects/{projectID}/evaluate", handleEvaluateFlags(evaluator))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/evaluate/{flagKey}", handleEvaluateFlag(evaluator))

	// Experiments
	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments", handleListExperiments(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments", handleCreateExperiment(db))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments/{experimentID}", handleGetExperiment(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments/{experimentID}/start", handleStartExperiment(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/experiments/{experimentID}/stop", handleStopExperiment(db))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/experiments/{experimentID}/results", handleExperimentResults(db))

	// Events
	mux.HandleFunc("POST /api/v1/events/exposures", handleIngestExposures())
	mux.HandleFunc("POST /api/v1/events/conversions", handleIngestConversions())

	// Audit
	mux.HandleFunc("GET /api/v1/projects/{projectID}/audit", handleListAuditLogs(db))

	// SSE streaming
	mux.HandleFunc("GET /api/v1/projects/{projectID}/stream/{envID}", handleSSEStream(sseHub))

	// Auth
	mux.HandleFunc("POST /api/v1/auth/login", handleLogin(db, authenticator))
	mux.HandleFunc("POST /api/v1/auth/register", handleRegister(db, authenticator))
	mux.HandleFunc("GET /api/v1/auth/me", handleMe())

	// Users
	mux.HandleFunc("GET /api/v1/projects/{projectID}/users", handleListProjectUsers(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/users", handleInviteUser(db, authenticator))
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/users/{userID}/role", handleUpdateUserRole(db))

	// GitOps
	mux.HandleFunc("POST /api/v1/projects/{projectID}/sync", handleGitOpsSync(db))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/diff", handleGitOpsDiff(db))

	// Chaos engineering
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

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

func handleListProjects(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := db.ListProjects(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if projects == nil {
			projects = []models.Project{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": projects, "total": len(projects)})
	}
}

func handleCreateProject(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Name == "" || req.Key == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("name and key are required"))
			return
		}
		p := &models.Project{Name: req.Name, Key: req.Key}
		if err := db.CreateProject(r.Context(), p); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, p)
	}
}

func handleGetProject(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := db.GetProject(r.Context(), r.PathValue("projectID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

// ---------------------------------------------------------------------------
// Environments
// ---------------------------------------------------------------------------

func handleListEnvironments(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		envs, err := db.ListEnvironments(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if envs == nil {
			envs = []models.Environment{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"environments": envs, "total": len(envs)})
	}
}

func handleCreateEnvironment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Key   string `json:"key"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Name == "" || req.Key == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("name and key are required"))
			return
		}
		e := &models.Environment{
			ProjectID: r.PathValue("projectID"),
			Name:      req.Name,
			Key:       req.Key,
			Color:     req.Color,
		}
		if err := db.CreateEnvironment(r.Context(), e); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, e)
	}
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

func handleListFlags(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := store.FlagFilter{
			Tag:    r.URL.Query().Get("tag"),
			Search: r.URL.Query().Get("search"),
		}
		if v := r.URL.Query().Get("enabled"); v != "" {
			b := v == "true"
			filter.Enabled = &b
		}
		if v := r.URL.Query().Get("archived"); v != "" {
			b := v == "true"
			filter.Archived = &b
		}

		flags, err := db.ListFlags(r.Context(), r.PathValue("projectID"), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if flags == nil {
			flags = []models.Flag{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"flags": flags, "total": len(flags)})
	}
}

func handleCreateFlag(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key          string          `json:"key"`
			Name         string          `json:"name"`
			Description  string          `json:"description"`
			Type         models.FlagType `json:"type"`
			DefaultValue json.RawMessage `json:"default_value"`
			Tags         []string        `json:"tags"`
			DependsOn    []string        `json:"depends_on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Key == "" || req.Name == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("key and name are required"))
			return
		}
		if req.Type == "" {
			req.Type = models.FlagTypeBoolean
		}
		if req.DefaultValue == nil {
			req.DefaultValue = json.RawMessage(`false`)
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}
		if req.DependsOn == nil {
			req.DependsOn = []string{}
		}

		f := &models.Flag{
			ProjectID:    r.PathValue("projectID"),
			Key:          req.Key,
			Name:         req.Name,
			Description:  req.Description,
			Type:         req.Type,
			DefaultValue: req.DefaultValue,
			Tags:         req.Tags,
			DependsOn:    req.DependsOn,
			CreatedBy:    getUserEmail(r),
		}
		if err := db.CreateFlag(r.Context(), f); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, f)
	}
}

func handleGetFlag(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, f)
	}
}

func handleUpdateFlag(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		var req struct {
			Name         *string          `json:"name"`
			Description  *string          `json:"description"`
			Tags         []string         `json:"tags"`
			DependsOn    []string         `json:"depends_on"`
			Archived     *bool            `json:"archived"`
			DefaultValue *json.RawMessage `json:"default_value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Name != nil {
			f.Name = *req.Name
		}
		if req.Description != nil {
			f.Description = *req.Description
		}
		if req.Tags != nil {
			f.Tags = req.Tags
		}
		if req.DependsOn != nil {
			f.DependsOn = req.DependsOn
		}
		if req.Archived != nil {
			f.Archived = *req.Archived
		}
		if req.DefaultValue != nil {
			f.DefaultValue = *req.DefaultValue
		}
		if err := db.UpdateFlag(r.Context(), f); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, f)
	}
}

func handleDeleteFlag(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		if err := db.DeleteFlag(r.Context(), f.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": f.ID})
	}
}

func handleToggleFlag(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		previousEnabled := f.Enabled
		f.Enabled = !f.Enabled

		if err := db.UpdateFlag(r.Context(), f); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:    f.ProjectID,
			Action:       models.AuditFlagToggled,
			ActorEmail:   getUserEmail(r),
			ResourceType: "flag",
			ResourceID:   f.ID,
			PreviousState: mustMarshal(map[string]bool{"enabled": previousEnabled}),
			NewState:      mustMarshal(map[string]bool{"enabled": f.Enabled}),
		})

		writeJSON(w, http.StatusOK, f)
	}
}

func handleKillSwitch(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		var req struct {
			Activate bool `json:"activate"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		f.KillSwitch = req.Activate
		if req.Activate {
			f.Enabled = false
		}

		if err := db.UpdateFlag(r.Context(), f); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:    f.ProjectID,
			Action:       models.AuditFlagKillSwitch,
			ActorEmail:   getUserEmail(r),
			ResourceType: "flag",
			ResourceID:   f.ID,
			NewState:     mustMarshal(map[string]bool{"kill_switch": f.KillSwitch, "enabled": f.Enabled}),
		})

		writeJSON(w, http.StatusOK, f)
	}
}

func handleRollback(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		f.Enabled = false
		f.KillSwitch = false

		if err := db.UpdateFlag(r.Context(), f); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:    f.ProjectID,
			Action:       models.AuditFlagRollback,
			ActorEmail:   getUserEmail(r),
			ResourceType: "flag",
			ResourceID:   f.ID,
		})

		writeJSON(w, http.StatusOK, f)
	}
}

// ---------------------------------------------------------------------------
// Flag-Environment configuration
// ---------------------------------------------------------------------------

func handleUpdateFlagEnvironment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		envID := r.PathValue("envID")
		var req struct {
			Enabled      *bool            `json:"enabled"`
			OffVariation *json.RawMessage `json:"off_variation"`
			Fallthrough  *models.RolloutConfig `json:"fallthrough"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		fe, err := db.GetFlagEnvironmentCtx(r.Context(), f.ID, envID)
		if store.IsNotFound(err) {
			// Create new flag-environment config
			fe = &models.FlagEnvironment{
				FlagID:        f.ID,
				EnvironmentID: envID,
			}
			if req.Enabled != nil {
				fe.Enabled = *req.Enabled
			}
			if req.OffVariation != nil {
				fe.OffVariation = *req.OffVariation
			}
			if req.Fallthrough != nil {
				fe.Fallthrough = *req.Fallthrough
			}
			if err := db.CreateFlagEnvironment(r.Context(), fe); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusCreated, fe)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if req.Enabled != nil {
			fe.Enabled = *req.Enabled
		}
		if req.OffVariation != nil {
			fe.OffVariation = *req.OffVariation
		}
		if req.Fallthrough != nil {
			fe.Fallthrough = *req.Fallthrough
		}

		if err := db.UpdateFlagEnvironment(r.Context(), fe); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, fe)
	}
}

func handleUpdateTargetingRules(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := db.GetFlagByKey(r.Context(), r.PathValue("projectID"), r.PathValue("flagKey"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		envID := r.PathValue("envID")
		var req struct {
			Rules []models.TargetingRule `json:"rules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		fe, err := db.GetFlagEnvironmentCtx(r.Context(), f.ID, envID)
		if err != nil {
			handleStoreError(w, err)
			return
		}

		fe.Rules = req.Rules
		if err := db.UpdateFlagEnvironment(r.Context(), fe); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:     f.ProjectID,
			EnvironmentID: envID,
			Action:        models.AuditRuleUpdated,
			ActorEmail:    getUserEmail(r),
			ResourceType:  "flag",
			ResourceID:    f.ID,
			NewState:      mustMarshal(req.Rules),
		})

		writeJSON(w, http.StatusOK, fe)
	}
}

// ---------------------------------------------------------------------------
// Evaluation
// ---------------------------------------------------------------------------

func handleEvaluateFlags(evaluator *flag.Evaluator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Context       models.EvalContext `json:"context"`
			EnvironmentID string            `json:"environment_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		results := evaluator.EvaluateAll(r.PathValue("projectID"), req.EnvironmentID, req.Context)
		writeJSON(w, http.StatusOK, map[string]any{"evaluations": results})
	}
}

func handleEvaluateFlag(evaluator *flag.Evaluator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Context       models.EvalContext `json:"context"`
			EnvironmentID string            `json:"environment_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		result := evaluator.Evaluate(r.PathValue("projectID"), req.EnvironmentID, r.PathValue("flagKey"), req.Context)
		writeJSON(w, http.StatusOK, result)
	}
}

// ---------------------------------------------------------------------------
// Experiments
// ---------------------------------------------------------------------------

func handleListExperiments(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exps, err := db.ListExperiments(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if exps == nil {
			exps = []models.Experiment{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"experiments": exps, "total": len(exps)})
	}
}

func handleCreateExperiment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exp models.Experiment
		if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		exp.ProjectID = r.PathValue("projectID")
		if exp.Status == "" {
			exp.Status = models.ExperimentDraft
		}
		if err := db.CreateExperiment(r.Context(), &exp); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, exp)
	}
}

func handleGetExperiment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, err := db.GetExperiment(r.Context(), r.PathValue("experimentID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, exp)
	}
}

func handleStartExperiment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, err := db.GetExperiment(r.Context(), r.PathValue("experimentID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		now := time.Now().UTC()
		exp.Status = models.ExperimentRunning
		exp.StartedAt = &now

		if err := db.UpdateExperiment(r.Context(), exp); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:    exp.ProjectID,
			Action:       models.AuditExperimentStart,
			ActorEmail:   getUserEmail(r),
			ResourceType: "experiment",
			ResourceID:   exp.ID,
		})

		writeJSON(w, http.StatusOK, exp)
	}
}

func handleStopExperiment(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, err := db.GetExperiment(r.Context(), r.PathValue("experimentID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		now := time.Now().UTC()
		exp.Status = models.ExperimentStopped
		exp.StoppedAt = &now

		if err := db.UpdateExperiment(r.Context(), exp); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		_ = db.WriteAuditLog(r.Context(), &models.AuditEntry{
			ProjectID:    exp.ProjectID,
			Action:       models.AuditExperimentStop,
			ActorEmail:   getUserEmail(r),
			ResourceType: "experiment",
			ResourceID:   exp.ID,
		})

		writeJSON(w, http.StatusOK, exp)
	}
}

func handleExperimentResults(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, err := db.GetExperiment(r.Context(), r.PathValue("experimentID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}
		// Return experiment with its current state — real analytics results
		// would come from the ClickHouse analytics pipeline
		writeJSON(w, http.StatusOK, map[string]any{
			"experiment_id": exp.ID,
			"status":        exp.Status,
			"started_at":    exp.StartedAt,
			"stopped_at":    exp.StoppedAt,
		})
	}
}

// ---------------------------------------------------------------------------
// Events (accepted and forwarded to ingestion pipeline)
// ---------------------------------------------------------------------------

func handleIngestExposures() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Events []models.ExposureEvent `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		// In production, these would be published to NATS for async ingestion
		writeJSON(w, http.StatusAccepted, map[string]int{"accepted": len(req.Events)})
	}
}

func handleIngestConversions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Events []models.ConversionEvent `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]int{"accepted": len(req.Events)})
	}
}

// ---------------------------------------------------------------------------
// Audit Logs
// ---------------------------------------------------------------------------

func handleListAuditLogs(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				offset = n
			}
		}

		entries, total, err := db.QueryAuditLogs(r.Context(), store.AuditLogQuery{
			ProjectID:    r.PathValue("projectID"),
			ResourceType: r.URL.Query().Get("resource_type"),
			ResourceID:   r.URL.Query().Get("resource_id"),
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if entries == nil {
			entries = []models.AuditEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "total": total})
	}
}

// ---------------------------------------------------------------------------
// SSE Streaming
// ---------------------------------------------------------------------------

func handleSSEStream(hub *streaming.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.Subscribe(r.PathValue("projectID"), r.PathValue("envID"), w, r)
	}
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func handleLogin(db *store.Store, authenticator *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email  string `json:"email"`
			APIKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		var user *models.User
		var err error
		if req.APIKey != "" {
			user, err = db.GetUserByAPIKey(r.Context(), req.APIKey)
		} else if req.Email != "" {
			user, err = db.GetUserByEmail(r.Context(), req.Email)
		} else {
			writeError(w, http.StatusBadRequest, fmt.Errorf("email or api_key required"))
			return
		}
		if err != nil {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
			return
		}

		token, err := authenticator.GenerateToken(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
	}
}

func handleRegister(db *store.Store, authenticator *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Email == "" || req.Name == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("email and name are required"))
			return
		}

		apiKey, err := auth.GenerateAPIKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		user := &models.User{
			Email:  req.Email,
			Name:   req.Name,
			Role:   models.RoleAdmin,
			APIKey: apiKey,
		}
		if err := db.CreateUser(r.Context(), user); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		token, err := authenticator.GenerateToken(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"token":   token,
			"user":    user,
			"api_key": apiKey,
		})
	}
}

func handleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUser(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func handleListProjectUsers(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := db.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if users == nil {
			users = []models.User{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users, "total": len(users)})
	}
}

func handleInviteUser(db *store.Store, authenticator *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string      `json:"email"`
			Name  string      `json:"name"`
			Role  models.Role `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Email == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("email is required"))
			return
		}
		if req.Role == "" {
			req.Role = models.RoleViewer
		}

		apiKey, err := auth.GenerateAPIKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		user := &models.User{
			Email:  req.Email,
			Name:   req.Name,
			Role:   req.Role,
			APIKey: apiKey,
		}
		if err := db.CreateUser(r.Context(), user); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		projectID := r.PathValue("projectID")
		if err := db.AddUserToProject(r.Context(), user.ID, projectID, req.Role); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"user": user, "api_key": apiKey})
	}
}

func handleUpdateUserRole(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Role models.Role `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}
		if req.Role == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("role is required"))
			return
		}

		user, err := db.GetUser(r.Context(), r.PathValue("userID"))
		if err != nil {
			handleStoreError(w, err)
			return
		}

		user.Role = req.Role
		if err := db.UpdateUser(r.Context(), user); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		projectID := r.PathValue("projectID")
		_ = db.AddUserToProject(r.Context(), user.ID, projectID, req.Role)

		writeJSON(w, http.StatusOK, user)
	}
}

// ---------------------------------------------------------------------------
// GitOps
// ---------------------------------------------------------------------------

func handleGitOpsSync(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Flags []struct {
				Key          string          `json:"key"`
				Name         string          `json:"name"`
				Description  string          `json:"description"`
				Type         models.FlagType `json:"type"`
				DefaultValue json.RawMessage `json:"default_value"`
				Tags         []string        `json:"tags"`
				Enabled      bool            `json:"enabled"`
			} `json:"flags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		projectID := r.PathValue("projectID")
		created, updated := 0, 0

		for _, rf := range req.Flags {
			existing, err := db.GetFlagByKey(r.Context(), projectID, rf.Key)
			if store.IsNotFound(err) {
				tags := rf.Tags
				if tags == nil {
					tags = []string{}
				}
				f := &models.Flag{
					ProjectID:    projectID,
					Key:          rf.Key,
					Name:         rf.Name,
					Description:  rf.Description,
					Type:         rf.Type,
					DefaultValue: rf.DefaultValue,
					Enabled:      rf.Enabled,
					Tags:         tags,
					DependsOn:    []string{},
					CreatedBy:    "gitops",
				}
				if err := db.CreateFlag(r.Context(), f); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				created++
			} else if err == nil {
				existing.Name = rf.Name
				existing.Description = rf.Description
				existing.Enabled = rf.Enabled
				if rf.Tags != nil {
					existing.Tags = rf.Tags
				}
				if err := db.UpdateFlag(r.Context(), existing); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				updated++
			} else {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"synced":  true,
			"created": created,
			"updated": updated,
			"total":   len(req.Flags),
		})
	}
}

func handleGitOpsDiff(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Flags []struct {
				Key     string `json:"key"`
				Enabled bool   `json:"enabled"`
			} `json:"flags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
			return
		}

		projectID := r.PathValue("projectID")
		var diffs []map[string]any

		for _, rf := range req.Flags {
			existing, err := db.GetFlagByKey(r.Context(), projectID, rf.Key)
			if store.IsNotFound(err) {
				diffs = append(diffs, map[string]any{
					"key":    rf.Key,
					"action": "create",
				})
			} else if err == nil {
				if existing.Enabled != rf.Enabled {
					diffs = append(diffs, map[string]any{
						"key":    rf.Key,
						"action": "update",
						"field":  "enabled",
						"from":   existing.Enabled,
						"to":     rf.Enabled,
					})
				}
			}
		}

		if diffs == nil {
			diffs = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"diffs": diffs})
	}
}

// ---------------------------------------------------------------------------
// Chaos Engineering
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("request error", "status", status, "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func handleStoreError(w http.ResponseWriter, err error) {
	var nfe *store.NotFoundError
	if errors.As(err, &nfe) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func getUserEmail(r *http.Request) string {
	user, ok := auth.GetUser(r.Context())
	if ok {
		return user.Email
	}
	return "anonymous"
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
