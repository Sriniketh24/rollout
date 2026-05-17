alter function public.create_default_workspace(text) security invoker;
revoke execute on function public.create_default_workspace(text) from public;
grant execute on function public.create_default_workspace(text) to authenticated;

drop policy if exists "authenticated users can read own profile" on public.users;
drop policy if exists "authenticated users can insert own profile" on public.users;
drop policy if exists "authenticated users can update own profile" on public.users;
create policy "authenticated users can read own profile"
  on public.users for select to authenticated
  using (id = (select auth.uid()));
create policy "authenticated users can insert own profile"
  on public.users for insert to authenticated
  with check (id = (select auth.uid()));
create policy "authenticated users can update own profile"
  on public.users for update to authenticated
  using (id = (select auth.uid()))
  with check (id = (select auth.uid()));

drop policy if exists "members can read projects" on public.projects;
drop policy if exists "authenticated users can create projects" on public.projects;
drop policy if exists "authenticated users can create own workspace project" on public.projects;
drop policy if exists "members can update projects" on public.projects;
create policy "members can read projects"
  on public.projects for select to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = (select auth.uid())
    )
  );
create policy "authenticated users can create own workspace project"
  on public.projects for insert to authenticated
  with check (
    key = 'workspace-' || left(replace((select auth.uid())::text, '-', ''), 12)
  );
create policy "members can update projects"
  on public.projects for update to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = projects.id and up.user_id = (select auth.uid())
    )
  );

drop policy if exists "users can read own project memberships" on public.user_projects;
drop policy if exists "users can insert own project memberships" on public.user_projects;
create policy "users can read own project memberships"
  on public.user_projects for select to authenticated
  using (user_id = (select auth.uid()));
create policy "users can insert own project memberships"
  on public.user_projects for insert to authenticated
  with check (user_id = (select auth.uid()));

drop policy if exists "members can manage environments" on public.environments;
create policy "members can manage environments"
  on public.environments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = environments.project_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = environments.project_id and up.user_id = (select auth.uid())
    )
  );

drop policy if exists "members can manage flags" on public.flags;
create policy "members can manage flags"
  on public.flags for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = flags.project_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = flags.project_id and up.user_id = (select auth.uid())
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
      where f.id = flag_environments.flag_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1
      from public.flags f
      join public.user_projects up on up.project_id = f.project_id
      where f.id = flag_environments.flag_id and up.user_id = (select auth.uid())
    )
  );

drop policy if exists "members can manage segments" on public.segments;
create policy "members can manage segments"
  on public.segments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = segments.project_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = segments.project_id and up.user_id = (select auth.uid())
    )
  );

drop policy if exists "members can manage experiments" on public.experiments;
create policy "members can manage experiments"
  on public.experiments for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = experiments.project_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = experiments.project_id and up.user_id = (select auth.uid())
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
      where e.id = experiment_results.experiment_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1
      from public.experiments e
      join public.user_projects up on up.project_id = e.project_id
      where e.id = experiment_results.experiment_id and up.user_id = (select auth.uid())
    )
  );

drop policy if exists "members can manage audit logs" on public.audit_logs;
create policy "members can manage audit logs"
  on public.audit_logs for all to authenticated
  using (
    exists (
      select 1 from public.user_projects up
      where up.project_id = audit_logs.project_id and up.user_id = (select auth.uid())
    )
  )
  with check (
    exists (
      select 1 from public.user_projects up
      where up.project_id = audit_logs.project_id and up.user_id = (select auth.uid())
    )
  );
