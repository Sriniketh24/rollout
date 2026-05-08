package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Sriniketh24/rollout/internal/models"
)

type Store interface {
	WriteAuditLog(ctx context.Context, entry *models.AuditEntry) error
	ListAuditLogs(ctx context.Context, projectID string, opts ListOptions) ([]models.AuditEntry, int64, error)
}

type ListOptions struct {
	EnvironmentID string
	ResourceType  string
	ResourceID    string
	ActorID       string
	Action        models.AuditAction
	Since         time.Time
	Until         time.Time
	Limit         int
	Offset        int
}

type Logger struct {
	store Store
}

func NewLogger(store Store) *Logger {
	return &Logger{store: store}
}

func (l *Logger) Log(ctx context.Context, entry *models.AuditEntry) error {
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	return l.store.WriteAuditLog(ctx, entry)
}

func (l *Logger) LogFlagChange(ctx context.Context, projectID, envID, actorID, actorEmail, flagID string, action models.AuditAction, prev, next any) error {
	prevJSON, _ := json.Marshal(prev)
	nextJSON, _ := json.Marshal(next)
	return l.Log(ctx, &models.AuditEntry{
		ProjectID:     projectID,
		EnvironmentID: envID,
		Action:        action,
		ActorID:       actorID,
		ActorEmail:    actorEmail,
		ResourceType:  "flag",
		ResourceID:    flagID,
		PreviousState: prevJSON,
		NewState:      nextJSON,
	})
}

func (l *Logger) LogExperiment(ctx context.Context, projectID, envID, actorID, actorEmail, experimentID string, action models.AuditAction) error {
	return l.Log(ctx, &models.AuditEntry{
		ProjectID:     projectID,
		EnvironmentID: envID,
		Action:        action,
		ActorID:       actorID,
		ActorEmail:    actorEmail,
		ResourceType:  "experiment",
		ResourceID:    experimentID,
	})
}

func (l *Logger) List(ctx context.Context, projectID string, opts ListOptions) ([]models.AuditEntry, int64, error) {
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	return l.store.ListAuditLogs(ctx, projectID, opts)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
