alter table public.flags
  add column variations jsonb not null default '[]'::jsonb,
  add column default_variation text not null default '';
