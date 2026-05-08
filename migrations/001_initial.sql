-- 001_initial.sql
-- Initial schema for Rollout feature flag platform.

-- Enable UUID generation.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Projects
-- ============================================================
CREATE TABLE projects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    key        TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_projects_key ON projects (key);

-- ============================================================
-- Environments
-- ============================================================
CREATE TABLE environments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    key        TEXT NOT NULL,
    color      TEXT NOT NULL DEFAULT '#6B7280',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE INDEX idx_environments_project_id ON environments (project_id);

-- ============================================================
-- Users
-- ============================================================
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'viewer',
    api_key    TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_api_key ON users (api_key);

-- ============================================================
-- User-Project many-to-many with role
-- ============================================================
CREATE TABLE user_projects (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX idx_user_projects_project_id ON user_projects (project_id);

-- ============================================================
-- Flags
-- ============================================================
CREATE TABLE flags (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key           TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    type          TEXT NOT NULL DEFAULT 'boolean',
    default_value JSONB NOT NULL DEFAULT 'false'::jsonb,
    enabled       BOOLEAN NOT NULL DEFAULT false,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    depends_on    TEXT[] NOT NULL DEFAULT '{}',
    kill_switch   BOOLEAN NOT NULL DEFAULT false,
    archived      BOOLEAN NOT NULL DEFAULT false,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE INDEX idx_flags_project_id ON flags (project_id);
CREATE INDEX idx_flags_project_key ON flags (project_id, key);
CREATE INDEX idx_flags_tags ON flags USING GIN (tags);

-- ============================================================
-- Flag-Environment join (per-env config)
-- ============================================================
CREATE TABLE flag_environments (
    flag_id        UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    enabled        BOOLEAN NOT NULL DEFAULT false,
    rules          JSONB NOT NULL DEFAULT '[]'::jsonb,
    fallthrough    JSONB NOT NULL DEFAULT '{"variations":[],"bucket_by":"key","seed":0}'::jsonb,
    off_variation   JSONB NOT NULL DEFAULT 'null'::jsonb,
    version        BIGINT NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (flag_id, environment_id)
);

CREATE INDEX idx_flag_environments_env_id ON flag_environments (environment_id);

-- ============================================================
-- Targeting Rules (denormalized in flag_environments.rules JSONB,
-- but kept as a table for querying / auditing individual rules)
-- ============================================================
CREATE TABLE targeting_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id          UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id   UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    clauses          JSONB NOT NULL DEFAULT '[]'::jsonb,
    variation        JSONB,
    rollout          JSONB,
    priority         INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_targeting_rules_flag_env ON targeting_rules (flag_id, environment_id);

-- ============================================================
-- Segments (reusable user segments)
-- ============================================================
CREATE TABLE segments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    rules       JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE INDEX idx_segments_project_id ON segments (project_id);

-- ============================================================
-- Audit Logs
-- ============================================================
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  UUID REFERENCES environments(id) ON DELETE SET NULL,
    action          TEXT NOT NULL,
    actor_id        UUID NOT NULL,
    actor_email     TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    previous_state  JSONB,
    new_state       JSONB,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_project_id ON audit_logs (project_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs (project_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource_type, resource_id);

-- ============================================================
-- Experiments
-- ============================================================
CREATE TABLE experiments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    flag_id           UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    environment_id    UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'draft',
    hypothesis        TEXT NOT NULL DEFAULT '',
    traffic_percent   INT NOT NULL DEFAULT 100,
    metrics           JSONB NOT NULL DEFAULT '[]'::jsonb,
    guardrail_metrics JSONB NOT NULL DEFAULT '[]'::jsonb,
    started_at        TIMESTAMPTZ,
    stopped_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_experiments_project_id ON experiments (project_id);
CREATE INDEX idx_experiments_flag_id ON experiments (flag_id);
CREATE INDEX idx_experiments_status ON experiments (status);
