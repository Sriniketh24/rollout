// ─── Flag Types ──────────────────────────────────────────────────────────────

export type FlagType = 'boolean' | 'string' | 'number' | 'json';

export interface Flag {
  id: string;
  project_id: string;
  key: string;
  name: string;
  description: string;
  type: FlagType;
  default_value: unknown;
  enabled: boolean;
  tags: string[];
  depends_on: string[];
  kill_switch: boolean;
  archived: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
  environments?: FlagEnvironment[];
}

export interface FlagEnvironment {
  flag_id: string;
  environment_id: string;
  enabled: boolean;
  rules: TargetingRule[];
  fallthrough: RolloutConfig;
  off_variation: unknown;
  version: number;
  updated_at: string;
}

// ─── Targeting ───────────────────────────────────────────────────────────────

export type Operator =
  | 'eq'
  | 'neq'
  | 'contains'
  | 'starts_with'
  | 'ends_with'
  | 'in'
  | 'not_in'
  | 'gt'
  | 'lt'
  | 'regex'
  | 'semver_gt'
  | 'semver_lt'
  | 'semver_eq';

export interface Clause {
  attribute: string;
  operator: Operator;
  values: string[];
  negate: boolean;
}

export interface TargetingRule {
  id: string;
  clauses: Clause[];
  variation?: unknown;
  rollout?: RolloutConfig;
  priority: number;
}

export interface RolloutConfig {
  variations: WeightedVariation[];
  bucket_by: string;
  seed: number;
}

export interface WeightedVariation {
  variation: unknown;
  weight: number; // basis points 0-10000
}

// ─── Evaluation ──────────────────────────────────────────────────────────────

export interface EvalContext {
  key: string;
  anonymous?: boolean;
  attributes?: Record<string, unknown>;
}

export type EvalReason =
  | 'TARGET_MATCH'
  | 'RULE_MATCH'
  | 'FALLTHROUGH'
  | 'OFF'
  | 'KILL_SWITCH'
  | 'ERROR'
  | 'DEFAULT'
  | 'DEPENDENCY_FAILED';

export interface EvalResult {
  flag_key: string;
  value: unknown;
  reason: EvalReason;
  version: number;
  timestamp: string;
}

// ─── Events ──────────────────────────────────────────────────────────────────

export interface ExposureEvent {
  flag_key: string;
  environment_id: string;
  user_key: string;
  variation: unknown;
  reason: EvalReason;
  timestamp: string;
}

// ─── SSE Events ──────────────────────────────────────────────────────────────

export type SSEEventType =
  | 'flag.updated'
  | 'flag.created'
  | 'flag.deleted'
  | 'flag.toggled'
  | 'environment.updated'
  | 'experiment.started'
  | 'experiment.stopped';

export interface SSEEvent {
  type: SSEEventType;
  flag_key?: string;
  data: unknown;
  version: number;
  timestamp: string;
}

// ─── Client Config ───────────────────────────────────────────────────────────

export interface RolloutClientConfig {
  apiKey: string;
  baseUrl: string;
  environment: string;
  /** Polling interval in milliseconds. Default: 30000 (30s) */
  pollingInterval?: number;
  /** Enable SSE streaming for real-time updates. Default: true */
  enableStreaming?: boolean;
  /** Interval in milliseconds to flush exposure events. Default: 10000 (10s) */
  flushInterval?: number;
  /** Maximum exposure events to batch before auto-flush. Default: 100 */
  maxBatchSize?: number;
}

// ─── Server Response Shapes ──────────────────────────────────────────────────

export interface FlagRulesResponse {
  flags: Flag[];
  flag_environments: FlagEnvironment[];
}
