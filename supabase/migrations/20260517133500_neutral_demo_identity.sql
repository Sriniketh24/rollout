update public.users
set
  email = 'demo-admin@rollout.dev',
  name = 'Demo Admin'
where id = '66666666-6666-6666-6666-666666666661';

update public.audit_logs
set actor_email = 'demo-admin@rollout.dev'
where actor_id = '66666666-6666-6666-6666-666666666661';
