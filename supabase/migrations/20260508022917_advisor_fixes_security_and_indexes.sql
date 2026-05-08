alter function public.set_updated_at() set search_path = public;

alter view public.dashboard_overview set (security_invoker = true);

create index idx_audit_logs_environment_id on public.audit_logs (environment_id);
create index idx_experiments_environment_id on public.experiments (environment_id);
create index idx_experiments_flag_id on public.experiments (flag_id);
create index idx_flags_created_by on public.flags (created_by);
