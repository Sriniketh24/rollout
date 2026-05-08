// ── Enums & Literal Types ──────────────────────────────────────────

export type FlagType = "boolean" | "string" | "number" | "json";

export type Operator =
  | "eq"
  | "neq"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "in"
  | "not_in"
  | "contains"
  | "not_contains"
  | "starts_with"
  | "ends_with"
  | "matches"
  | "exists"
  | "not_exists"
  | "semver_eq"
  | "semver_gt"
  | "semver_lt";

export type ExperimentStatus =
  | "draft"
  | "running"
  | "paused"
  | "stopped"
  | "complete";

export type AuditAction =
  | "flag.created"
  | "flag.updated"
  | "flag.deleted"
  | "flag.toggled"
  | "flag.killed"
  | "experiment.created"
  | "experiment.started"
  | "experiment.paused"
  | "experiment.stopped"
  | "experiment.completed"
  | "environment.created"
  | "environment.updated"
  | "environment.deleted"
  | "targeting.updated"
  | "rollout.updated"
  | "project.updated"
  | "user.invited"
  | "user.removed";

export type UserRole = "owner" | "admin" | "editor" | "viewer";

// ── Core Models ────────────────────────────────────────────────────

export interface Clause {
  attribute: string;
  operator: Operator;
  values: string[];
  negate?: boolean;
}

export interface TargetingRule {
  id: string;
  description?: string;
  clauses: Clause[];
  variation: string;
  rolloutPercentage: number;
  priority: number;
}

export interface FlagVariation {
  key: string;
  value: unknown;
  name?: string;
  description?: string;
}

export interface EnvironmentConfig {
  environmentId: string;
  enabled: boolean;
  rolloutPercentage: number;
  targetingRules: TargetingRule[];
  offVariation?: string;
  killSwitchActive?: boolean;
  lastModified: string;
}

export interface Flag {
  id: string;
  key: string;
  name: string;
  description: string;
  type: FlagType;
  tags: string[];
  variations: FlagVariation[];
  defaultVariation: string;
  environments: Record<string, EnvironmentConfig>;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  archived: boolean;
  projectId: string;
}

export interface Environment {
  id: string;
  key: string;
  name: string;
  color: string;
  description?: string;
  production: boolean;
  createdAt: string;
  updatedAt: string;
  projectId: string;
  order: number;
}

export interface ExperimentVariation {
  key: string;
  name?: string;
  weight: number;
  isControl: boolean;
}

export interface GuardrailMetric {
  key: string;
  name: string;
  type: "conversion" | "count" | "revenue";
  minimumDetectableEffect: number;
}

export interface Experiment {
  id: string;
  key: string;
  name: string;
  description: string;
  flagId: string;
  flagKey?: string;
  status: ExperimentStatus;
  variations: ExperimentVariation[];
  trafficPercentage: number;
  startDate?: string;
  endDate?: string;
  targetMetric: string;
  guardrailMetrics: GuardrailMetric[];
  createdAt: string;
  updatedAt: string;
  projectId: string;
}

export interface EvalContext {
  userId?: string;
  attributes: Record<string, unknown>;
  environmentKey: string;
}

export interface EvalResult {
  flagKey: string;
  variation: string;
  value: unknown;
  reason: string;
  ruleId?: string;
  experimentId?: string;
}

export interface VariationResult {
  variationKey: string;
  name?: string;
  isControl: boolean;
  sampleSize: number;
  conversions: number;
  conversionRate: number;
  improvementOverControl: number;
  probabilityToBeatControl: number;
  credibleInterval: {
    lower: number;
    upper: number;
  };
  isWinner: boolean;
  isStatisticallySignificant: boolean;
}

export interface GuardrailResult {
  metricKey: string;
  metricName: string;
  passed: boolean;
  controlValue: number;
  treatmentValue: number;
  pValue: number;
  message: string;
}

export interface AIRecommendation {
  action: "ship" | "iterate" | "stop" | "extend";
  confidence: number;
  reasoning: string;
  suggestedNextSteps: string[];
}

export interface ExperimentResults {
  experimentId: string;
  status: ExperimentStatus;
  totalSampleSize: number;
  startDate: string;
  endDate?: string;
  variationResults: VariationResult[];
  guardrailResults: GuardrailResult[];
  recommendation?: AIRecommendation;
  lastUpdated: string;
}

export interface AuditEntry {
  id: string;
  action: AuditAction;
  actor: string;
  actorEmail?: string;
  targetType: string;
  targetId: string;
  targetName?: string;
  timestamp: string;
  changes?: Record<string, { before: unknown; after: unknown }>;
  metadata?: Record<string, unknown>;
  projectId: string;
  environmentId?: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  avatarUrl?: string;
  lastLogin?: string;
  createdAt: string;
}

export interface Project {
  id: string;
  key: string;
  name: string;
  description?: string;
  apiKey: string;
  serverApiKey: string;
  createdAt: string;
  updatedAt: string;
}

// ── API Response Wrappers ──────────────────────────────────────────

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, string>;
}

// ── SSE Event Types ────────────────────────────────────────────────

export interface SSEEvent {
  type:
    | "flag.updated"
    | "flag.deleted"
    | "experiment.updated"
    | "environment.updated"
    | "audit.created";
  data: unknown;
  timestamp: string;
}
