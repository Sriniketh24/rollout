update public.users
set
  email = 'demo-admin@rollout.dev',
  name = 'Demo Admin'
where email = 'sriniketh@example.com' or name = 'Sriniketh';

update public.audit_logs
set actor_email = 'demo-admin@rollout.dev'
where actor_email = 'sriniketh@example.com';
