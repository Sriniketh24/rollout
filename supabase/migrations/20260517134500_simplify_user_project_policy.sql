drop policy if exists "members can read project memberships" on public.user_projects;

create policy "users can read own project memberships"
  on public.user_projects for select to authenticated
  using (user_id = auth.uid());
