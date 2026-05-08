package models

import (
	"encoding/json"
	"time"
)

type Environment struct {
	ID        string    `json:"id" db:"id"`
	ProjectID string    `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	Key       string    `json:"key" db:"key"`
	Color     string    `json:"color" db:"color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

type Flag struct {
	ID            string          `json:"id" db:"id"`
	ProjectID     string          `json:"project_id" db:"project_id"`
	Key           string          `json:"key" db:"key"`
	Name          string          `json:"name" db:"name"`
	Description   string          `json:"description" db:"description"`
	Type          FlagType        `json:"type" db:"type"`
	DefaultValue  json.RawMessage `json:"default_value" db:"default_value"`
	Enabled       bool            `json:"enabled" db:"enabled"`
	Tags          []string        `json:"tags" db:"tags"`
	DependsOn     []string        `json:"depends_on" db:"depends_on"`
	KillSwitch    bool            `json:"kill_switch" db:"kill_switch"`
	Archived      bool            `json:"archived" db:"archived"`
	CreatedBy     string          `json:"created_by" db:"created_by"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	Environments  []FlagEnvironment `json:"environments,omitempty"`
}

type FlagEnvironment struct {
	FlagID        string           `json:"flag_id" db:"flag_id"`
	EnvironmentID string           `json:"environment_id" db:"environment_id"`
	Enabled       bool             `json:"enabled" db:"enabled"`
	Rules         []TargetingRule  `json:"rules" db:"rules"`
	Fallthrough   RolloutConfig    `json:"fallthrough" db:"fallthrough"`
	OffVariation  json.RawMessage  `json:"off_variation" db:"off_variation"`
	Version       int64            `json:"version" db:"version"`
	UpdatedAt     time.Time        `json:"updated_at" db:"updated_at"`
}

type Operator string

const (
	OpEquals      Operator = "eq"
	OpNotEquals   Operator = "neq"
	OpContains    Operator = "contains"
	OpStartsWith  Operator = "starts_with"
	OpEndsWith    Operator = "ends_with"
	OpIn          Operator = "in"
	OpNotIn       Operator = "not_in"
	OpGreaterThan Operator = "gt"
	OpLessThan    Operator = "lt"
	OpRegex       Operator = "regex"
	OpSemverGT    Operator = "semver_gt"
	OpSemverLT    Operator = "semver_lt"
	OpSemverEQ    Operator = "semver_eq"
)

type TargetingRule struct {
	ID         string          `json:"id"`
	Clauses    []Clause        `json:"clauses"`
	Variation  json.RawMessage `json:"variation,omitempty"`
	Rollout    *RolloutConfig  `json:"rollout,omitempty"`
	Priority   int             `json:"priority"`
}

type Clause struct {
	Attribute string   `json:"attribute"`
	Operator  Operator `json:"operator"`
	Values    []string `json:"values"`
	Negate    bool     `json:"negate"`
}

type RolloutConfig struct {
	Variations []WeightedVariation `json:"variations"`
	BucketBy   string              `json:"bucket_by"`
	Seed       int64               `json:"seed"`
}

type WeightedVariation struct {
	Variation json.RawMessage `json:"variation"`
	Weight    int             `json:"weight"` // basis points (0-10000)
}

type EvalContext struct {
	Key        string            `json:"key"`
	Anonymous  bool              `json:"anonymous,omitempty"`
	Attributes map[string]any    `json:"attributes,omitempty"`
}

type EvalResult struct {
	FlagKey   string          `json:"flag_key"`
	Value     json.RawMessage `json:"value"`
	Reason    EvalReason      `json:"reason"`
	Version   int64           `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
}

type EvalReason string

const (
	ReasonTargetMatch  EvalReason = "TARGET_MATCH"
	ReasonRuleMatch    EvalReason = "RULE_MATCH"
	ReasonFallthrough  EvalReason = "FALLTHROUGH"
	ReasonOff          EvalReason = "OFF"
	ReasonKillSwitch   EvalReason = "KILL_SWITCH"
	ReasonError        EvalReason = "ERROR"
	ReasonDefault      EvalReason = "DEFAULT"
	ReasonDependency   EvalReason = "DEPENDENCY_FAILED"
)

// RBAC

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	Role      Role      `json:"role" db:"role"`
	APIKey    string    `json:"-" db:"api_key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Project struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Key       string    `json:"key" db:"key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Audit

type AuditAction string

const (
	AuditFlagCreated     AuditAction = "flag.created"
	AuditFlagUpdated     AuditAction = "flag.updated"
	AuditFlagDeleted     AuditAction = "flag.deleted"
	AuditFlagToggled     AuditAction = "flag.toggled"
	AuditFlagKillSwitch  AuditAction = "flag.kill_switch"
	AuditFlagRollback    AuditAction = "flag.rollback"
	AuditRuleCreated     AuditAction = "rule.created"
	AuditRuleUpdated     AuditAction = "rule.updated"
	AuditRuleDeleted     AuditAction = "rule.deleted"
	AuditEnvCreated      AuditAction = "env.created"
	AuditUserCreated     AuditAction = "user.created"
	AuditUserRoleChanged AuditAction = "user.role_changed"
	AuditExperimentStart AuditAction = "experiment.started"
	AuditExperimentStop  AuditAction = "experiment.stopped"
)

type AuditEntry struct {
	ID            string          `json:"id" db:"id"`
	ProjectID     string          `json:"project_id" db:"project_id"`
	EnvironmentID string          `json:"environment_id" db:"environment_id"`
	Action        AuditAction     `json:"action" db:"action"`
	ActorID       string          `json:"actor_id" db:"actor_id"`
	ActorEmail    string          `json:"actor_email" db:"actor_email"`
	ResourceType  string          `json:"resource_type" db:"resource_type"`
	ResourceID    string          `json:"resource_id" db:"resource_id"`
	PreviousState json.RawMessage `json:"previous_state,omitempty" db:"previous_state"`
	NewState      json.RawMessage `json:"new_state,omitempty" db:"new_state"`
	Timestamp     time.Time       `json:"timestamp" db:"timestamp"`
}

// Experimentation

type ExperimentStatus string

const (
	ExperimentDraft    ExperimentStatus = "draft"
	ExperimentRunning  ExperimentStatus = "running"
	ExperimentPaused   ExperimentStatus = "paused"
	ExperimentStopped  ExperimentStatus = "stopped"
	ExperimentComplete ExperimentStatus = "complete"
)

type Experiment struct {
	ID              string           `json:"id" db:"id"`
	ProjectID       string           `json:"project_id" db:"project_id"`
	FlagID          string           `json:"flag_id" db:"flag_id"`
	EnvironmentID   string           `json:"environment_id" db:"environment_id"`
	Name            string           `json:"name" db:"name"`
	Description     string           `json:"description" db:"description"`
	Status          ExperimentStatus `json:"status" db:"status"`
	Hypothesis      string           `json:"hypothesis" db:"hypothesis"`
	TrafficPercent  int              `json:"traffic_percent" db:"traffic_percent"`
	Metrics         []Metric         `json:"metrics" db:"metrics"`
	GuardrailMetrics []Metric        `json:"guardrail_metrics" db:"guardrail_metrics"`
	StartedAt       *time.Time       `json:"started_at,omitempty" db:"started_at"`
	StoppedAt       *time.Time       `json:"stopped_at,omitempty" db:"stopped_at"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

type MetricType string

const (
	MetricConversion MetricType = "conversion"
	MetricRevenue    MetricType = "revenue"
	MetricCount      MetricType = "count"
	MetricDuration   MetricType = "duration"
)

type Metric struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Type        MetricType `json:"type"`
	EventName   string     `json:"event_name"`
	IsGuardrail bool       `json:"is_guardrail"`
}

// Events

type ExposureEvent struct {
	FlagKey       string          `json:"flag_key"`
	EnvironmentID string          `json:"environment_id"`
	UserKey       string          `json:"user_key"`
	Variation     json.RawMessage `json:"variation"`
	Reason        EvalReason      `json:"reason"`
	Timestamp     time.Time       `json:"timestamp"`
}

type ConversionEvent struct {
	ExperimentID  string          `json:"experiment_id"`
	EnvironmentID string          `json:"environment_id"`
	UserKey       string          `json:"user_key"`
	MetricKey     string          `json:"metric_key"`
	Value         float64         `json:"value"`
	Properties    json.RawMessage `json:"properties,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// Experiment Results (Bayesian)

type ExperimentResults struct {
	ExperimentID       string             `json:"experiment_id"`
	ComputedAt         time.Time          `json:"computed_at"`
	TotalExposures     int64              `json:"total_exposures"`
	VariationResults   []VariationResult  `json:"variation_results"`
	Recommendation     string             `json:"recommendation"`
	AIInsight          string             `json:"ai_insight,omitempty"`
}

type VariationResult struct {
	VariationKey       string             `json:"variation_key"`
	Exposures          int64              `json:"exposures"`
	Conversions        int64              `json:"conversions"`
	ConversionRate     float64            `json:"conversion_rate"`
	ProbabilityToBeat  float64            `json:"probability_to_beat_control"`
	ExpectedLoss       float64            `json:"expected_loss"`
	CredibleInterval   [2]float64         `json:"credible_interval_95"`
	IsControl          bool               `json:"is_control"`
	GuardrailResults   []GuardrailResult  `json:"guardrail_results"`
}

type GuardrailResult struct {
	MetricKey       string     `json:"metric_key"`
	ControlMean     float64    `json:"control_mean"`
	VariantMean     float64    `json:"variant_mean"`
	RelativeChange  float64    `json:"relative_change"`
	Degraded        bool       `json:"degraded"`
}

// Health

type ServiceHealth struct {
	Service   string            `json:"service"`
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}
