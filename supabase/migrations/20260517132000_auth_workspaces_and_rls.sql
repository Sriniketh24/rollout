create or replace function public.create_default_workspace(display_name text default null)
returns uuid
language plpgsql
security invoker
set search_path = public
as $$
declare
  current_user_id uuid := auth.uid();
  user_email text := coalesce(auth.jwt() ->> 'email', 'user@example.com');
  profile_name text := coalesce(
    nullif(display_name, ''),
    nullif(auth.jwt() -> 'user_metadata' ->> 'name', ''),
    nullif(split_part(user_email, '@', 1), ''),
    'User'
  );
  existing_project_id uuid;
  new_project_id uuid;
begin
  if current_user_id is null then
    raise exception 'create_default_workspace requires an authenticated user';
  end if;

  select up.project_id
  into existing_project_id
  from public.user_projects up
  where up.user_id = current_user_id
  order by up.created_at asc
  limit 1;

  if existing_project_id is not null then
    return existing_project_id;
  end if;

  insert into public.users (id, email, name, role, api_key)
  values (
    current_user_id,
    user_email,
    profile_name,
    'owner',
    'rol_' || replace(current_user_id::text, '-', '')
  )
  on conflict (id) do update
  set
    email = excluded.email,
    name = excluded.name
  returning id into current_user_id;

  insert into public.projects (name, key)
  values (
    profile_name || '''s Workspace',
    'workspace-' || left(replace(current_user_id::text, '-', ''), 12)
  )
  returning id into new_project_id;

  insert into public.user_projects (user_id, project_id, role)
  values (current_user_id, new_project_id, 'owner');

  insert into public.environments (
    project_id,
    key,
    name,
    color,
    description,
    production,
    sort_order
  )
  values
    (new_project_id, 'development', 'Development', '#3b82f6', 'Local development environment', false, 0),
    (new_project_id, 'staging', 'Staging', '#eab308', 'Pre-production testing', false, 1),
    (new_project_id, 'production', 'Production', '#ef4444', 'Live production environment', true, 2);

  insert into public.audit_logs (
    project_id,
    action,
    actor_id,
    actor_email,
    resource_type,
    resource_id,
    previous_state,
    new_state
  )
  values (
    new_project_id,
    'project.updated',
    current_user_id,
    user_email,
    'project',
    new_project_id::text,
    null,
    jsonb_build_object('name', profile_name || '''s Workspace')
  );

  return new_project_id;
end;
$$;

grant execute on function public.create_default_workspace(text) to authenticated;

drop policy if exists "authenticated users can read own profile" on public.users;
drop policy if exists "authenticated users can insert own profile" on public.users;
drop policy if exists "authenticated users can update own profile" on public.users;
create policy "authenticated users can read own profile"
  on public.users for select to authenticated
  using (id = auth.uid());
create policy "authenticated users can insert own profile"
  on public.users for insert to authenticated
  with check (id = auth.uid());
create policy "authenticated users can update own profile"
  on public.users for update to authenticated
  using (id = auth.uid())
  with check (id = auth.uid());

drop policy if exists "members can read projects" on public.projects;
drop policy if exists "authenticated users can create projects" on public.projects;
drop policy if exists "members can update projects" on public.projects;
create policy "members can read projects"
  on public.projects for select to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = auth.uid()
    )
  );
create policy "authenticated users can create projects"
  on public.projects for insert to authenticated
  with check (true);
create policy "members can update projects"
  on public.projects for update to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can read project memberships" on public.user_projects;
drop policy if exists "users can insert own project memberships" on public.user_projects;
create policy "users can read own project memberships"
  on public.user_projects for select to authenticated
  using (user_id = auth.uid());
create policy "users can insert own project memberships"
  on public.user_projects for insert to authenticated
  with check (user_id = auth.uid());

drop policy if exists "members can manage environments" on public.environments;
create policy "members can manage environments"
  on public.environments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = environments.project_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = environments.project_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage flags" on public.flags;
create policy "members can manage flags"
  on public.flags for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = flags.project_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = flags.project_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage flag environments" on public.flag_environments;
create policy "members can manage flag environments"
  on public.flag_environments for all to authenticated
  using (
    exists (
      select 1
      from public.flags f
      join public.user_projects up on up.project_id = f.project_id
      where f.id = flag_environments.flag_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1
      from public.flags f
      join public.user_projects up on up.project_id = f.project_id
      where f.id = flag_environments.flag_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage segments" on public.segments;
create policy "members can manage segments"
  on public.segments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = segments.project_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = segments.project_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage experiments" on public.experiments;
create policy "members can manage experiments"
  on public.experiments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = experiments.project_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = experiments.project_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage experiment results" on public.experiment_results;
create policy "members can manage experiment results"
  on public.experiment_results for all to authenticated
  using (
    exists (
      select 1
      from public.experiments e
      join public.user_projects up on up.project_id = e.project_id
      where e.id = experiment_results.experiment_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1
      from public.experiments e
      join public.user_projects up on up.project_id = e.project_id
      where e.id = experiment_results.experiment_id and up.user_id = auth.uid()
    )
  );

drop policy if exists "members can manage audit logs" on public.audit_logs;
create policy "members can manage audit logs"
  on public.audit_logs for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = audit_logs.project_id and up.user_id = auth.uid()
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = audit_logs.project_id and up.user_id = auth.uid()
    )
  );
