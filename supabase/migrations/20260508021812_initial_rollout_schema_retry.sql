create extension if not exists pgcrypto;

create table public.projects (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  key text not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.users (
  id uuid primary key default gen_random_uuid(),
  email text not null unique,
  name text not null,
  role text not null default 'viewer',
  api_key text not null unique,
  created_at timestamptz not null default now()
);

create table public.user_projects (
  user_id uuid not null references public.users (id) on delete cascade,
  project_id uuid not null references public.projects (id) on delete cascade,
  role text not null default 'viewer',
  created_at timestamptz not null default now(),
  primary key (user_id, project_id)
);

create table public.environments (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects (id) on delete cascade,
  name text not null,
  key text not null,
  color text not null default '#6B7280',
  description text not null default '',
  production boolean not null default false,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (project_id, key)
);

create table public.flags (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects (id) on delete cascade,
  key text not null,
  name text not null,
  description text not null default '',
  type text not null default 'boolean',
  default_value jsonb not null default 'false'::jsonb,
  enabled boolean not null default false,
  tags text[] not null default '{}'::text[],
  depends_on text[] not null default '{}'::text[],
  kill_switch boolean not null default false,
  archived boolean not null default false,
  created_by uuid references public.users (id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (project_id, key)
);

create table public.flag_environments (
  flag_id uuid not null references public.flags (id) on delete cascade,
  environment_id uuid not null references public.environments (id) on delete cascade,
  enabled boolean not null default false,
  rules jsonb not null default '[]'::jsonb,
  fallthrough jsonb not null default '{"seed": 0, "bucket_by": "key", "variations": []}'::jsonb,
  off_variation jsonb not null default 'null'::jsonb,
  rollout_percentage integer not null default 0,
  kill_switch_active boolean not null default false,
  version bigint not null default 1,
  updated_at timestamptz not null default now(),
  primary key (flag_id, environment_id)
);

create table public.segments (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects (id) on delete cascade,
  key text not null,
  name text not null,
  description text not null default '',
  rules jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (project_id, key)
);

create table public.audit_logs (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects (id) on delete cascade,
  environment_id uuid references public.environments (id) on delete set null,
  action text not null,
  actor_id uuid not null,
  actor_email text not null,
  resource_type text not null,
  resource_id text not null,
  previous_state jsonb,
  new_state jsonb,
  timestamp timestamptz not null default now()
);

create table public.experiments (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects (id) on delete cascade,
  flag_id uuid not null references public.flags (id) on delete cascade,
  environment_id uuid not null references public.environments (id) on delete cascade,
  key text not null,
  name text not null,
  description text not null default '',
  status text not null default 'draft',
  hypothesis text not null default '',
  traffic_percent integer not null default 100,
  target_metric text not null default 'conversion_rate',
  metrics jsonb not null default '[]'::jsonb,
  guardrail_metrics jsonb not null default '[]'::jsonb,
  variations jsonb not null default '[]'::jsonb,
  started_at timestamptz,
  stopped_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (project_id, key)
);

create table public.experiment_results (
  experiment_id uuid primary key references public.experiments (id) on delete cascade,
  status text not null default 'draft',
  total_sample_size integer not null default 0,
  start_date timestamptz not null default now(),
  end_date timestamptz,
  variation_results jsonb not null default '[]'::jsonb,
  guardrail_results jsonb not null default '[]'::jsonb,
  recommendation jsonb,
  last_updated timestamptz not null default now()
);

create index idx_audit_logs_project_id on public.audit_logs (project_id);
create index idx_audit_logs_timestamp on public.audit_logs (project_id, "timestamp" desc);
create index idx_environments_project_id on public.environments (project_id);
create index idx_experiments_project_id on public.experiments (project_id);
create index idx_experiments_status on public.experiments (status);
create index idx_flag_environments_env_id on public.flag_environments (environment_id);
create index idx_flags_project_id on public.flags (project_id);
create index idx_flags_project_key on public.flags (project_id, key);
create index idx_flags_tags on public.flags using gin (tags);
create index idx_segments_project_id on public.segments (project_id);
create index idx_user_projects_project_id on public.user_projects (project_id);

create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger set_projects_updated_at
before update on public.projects
for each row execute function public.set_updated_at();

create trigger set_environments_updated_at
before update on public.environments
for each row execute function public.set_updated_at();

create trigger set_flags_updated_at
before update on public.flags
for each row execute function public.set_updated_at();

create trigger set_segments_updated_at
before update on public.segments
for each row execute function public.set_updated_at();

create trigger set_experiments_updated_at
before update on public.experiments
for each row execute function public.set_updated_at();

create view public.dashboard_overview as
select
  p.id as project_id,
  p.key as project_key,
  (
    select count(*)
    from public.flags f
    where f.project_id = p.id and f.archived = false
  ) as flag_count,
  (
    select count(*)
    from public.experiments e
    where e.project_id = p.id and e.status = 'running'
  ) as active_experiment_count,
  (
    select count(*)
    from public.environments env
    where env.project_id = p.id
  ) as environment_count,
  (
    select count(*)
    from public.audit_logs a
    where a.project_id = p.id and a."timestamp" >= now() - interval '24 hours'
  ) as audit_events_24h
from public.projects p;

alter table public.projects enable row level security;
alter table public.users enable row level security;
alter table public.user_projects enable row level security;
alter table public.environments enable row level security;
alter table public.flags enable row level security;
alter table public.flag_environments enable row level security;
alter table public.segments enable row level security;
alter table public.audit_logs enable row level security;
alter table public.experiments enable row level security;
alter table public.experiment_results enable row level security;

create policy deny_anon_projects on public.projects for all to anon using (false) with check (false);
create policy deny_anon_users on public.users for all to anon using (false) with check (false);
create policy deny_anon_user_projects on public.user_projects for all to anon using (false) with check (false);
create policy deny_anon_environments on public.environments for all to anon using (false) with check (false);
create policy deny_anon_flags on public.flags for all to anon using (false) with check (false);
create policy deny_anon_flag_environments on public.flag_environments for all to anon using (false) with check (false);
create policy deny_anon_segments on public.segments for all to anon using (false) with check (false);
create policy deny_anon_audit_logs on public.audit_logs for all to anon using (false) with check (false);
create policy deny_anon_experiments on public.experiments for all to anon using (false) with check (false);
create policy deny_anon_experiment_results on public.experiment_results for all to anon using (false) with check (false);
