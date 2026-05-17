create table if not exists public.guest_workspace_tokens (
  token_hash text primary key,
  project_id uuid not null unique references public.projects (id) on delete cascade,
  created_at timestamptz not null default now(),
  last_used_at timestamptz not null default now()
);

alter table public.guest_workspace_tokens enable row level security;

drop policy if exists deny_anon_guest_workspace_tokens on public.guest_workspace_tokens;
drop policy if exists deny_authenticated_guest_workspace_tokens on public.guest_workspace_tokens;
create policy deny_anon_guest_workspace_tokens
  on public.guest_workspace_tokens for all to anon
  using (false)
  with check (false);
create policy deny_authenticated_guest_workspace_tokens
  on public.guest_workspace_tokens for all to authenticated
  using (false)
  with check (false);
