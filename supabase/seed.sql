insert into public.users (id, email, name, role, api_key, created_at)
values
  ('66666666-6666-6666-6666-666666666661', 'demo-admin@rollout.dev', 'Demo Admin', 'owner', 'rol_demo_owner_key', '2026-05-08T02:19:54.645629+00:00'),
  ('66666666-6666-6666-6666-666666666662', 'pm@example.com', 'Ava Product', 'editor', 'rol_demo_pm_key', '2026-05-08T02:19:54.645629+00:00')
on conflict (id) do update
set
  email = excluded.email,
  name = excluded.name,
  role = excluded.role,
  api_key = excluded.api_key;

insert into public.projects (id, key, name, created_at, updated_at)
values
  ('11111111-1111-1111-1111-111111111111', 'rollout-demo', 'Rollout Demo', '2026-05-08T02:19:54.645629+00:00', '2026-05-08T02:19:54.645629+00:00')
on conflict (id) do update
set
  key = excluded.key,
  name = excluded.name,
  updated_at = excluded.updated_at;

insert into public.user_projects (user_id, project_id, role, created_at)
values
  ('66666666-6666-6666-6666-666666666661', '11111111-1111-1111-1111-111111111111', 'owner', '2026-05-08T02:19:54.645629+00:00'),
  ('66666666-6666-6666-6666-666666666662', '11111111-1111-1111-1111-111111111111', 'editor', '2026-05-08T02:19:54.645629+00:00')
on conflict (user_id, project_id) do update
set role = excluded.role;

insert into public.environments (
  id,
  project_id,
  key,
  name,
  color,
  description,
  production,
  sort_order,
  created_at,
  updated_at
)
values
  ('22222222-2222-2222-2222-222222222221', '11111111-1111-1111-1111-111111111111', 'development', 'Development', '#3b82f6', 'Local development environment', false, 0, '2026-05-08T02:19:54.645629+00:00', '2026-05-08T02:19:54.645629+00:00'),
  ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'staging', 'Staging', '#eab308', 'Pre-production testing', false, 1, '2026-05-08T02:19:54.645629+00:00', '2026-05-08T02:19:54.645629+00:00'),
  ('22222222-2222-2222-2222-222222222223', '11111111-1111-1111-1111-111111111111', 'production', 'Production', '#ef4444', 'Live production environment', true, 2, '2026-05-08T02:19:54.645629+00:00', '2026-05-08T02:19:54.645629+00:00')
on conflict (id) do update
set
  name = excluded.name,
  color = excluded.color,
  description = excluded.description,
  production = excluded.production,
  sort_order = excluded.sort_order,
  updated_at = excluded.updated_at;

insert into public.flags (
  id,
  project_id,
  key,
  name,
  description,
  type,
  default_value,
  enabled,
  tags,
  depends_on,
  kill_switch,
  archived,
  created_by,
  created_at,
  updated_at,
  variations,
  default_variation
)
values
  (
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    'ai-recommendations',
    'AI Recommendations',
    'ML-powered product recommendations engine',
    'string',
    '"none"'::jsonb,
    false,
    array['ml', 'backend', 'performance'],
    '{}'::text[],
    false,
    false,
    '66666666-6666-6666-6666-666666666662',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:21:49.349972+00:00',
    '[
      {"key":"v1","name":"Collaborative Filtering","value":"collaborative-filtering"},
      {"key":"v2","name":"Neural Network","value":"neural-network"},
      {"key":"off","name":"Disabled","value":"none"}
    ]'::jsonb,
    'off'
  ),
  (
    '33333333-3333-3333-3333-333333333331',
    '11111111-1111-1111-1111-111111111111',
    'dark-mode',
    'Dark Mode',
    'Enable dark mode across the application',
    'boolean',
    'false'::jsonb,
    true,
    array['ui', 'frontend'],
    '{}'::text[],
    false,
    false,
    '66666666-6666-6666-6666-666666666661',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:21:49.349972+00:00',
    '[
      {"key":"on","name":"Enabled","value":true},
      {"key":"off","name":"Disabled","value":false}
    ]'::jsonb,
    'off'
  ),
  (
    '33333333-3333-3333-3333-333333333332',
    '11111111-1111-1111-1111-111111111111',
    'new-checkout-flow',
    'New Checkout Flow',
    'Redesigned checkout experience with fewer steps',
    'boolean',
    'false'::jsonb,
    false,
    array['checkout', 'experiment'],
    '{}'::text[],
    false,
    false,
    '66666666-6666-6666-6666-666666666661',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:21:49.349972+00:00',
    '[
      {"key":"on","name":"New Flow","value":true},
      {"key":"off","name":"Legacy Flow","value":false}
    ]'::jsonb,
    'off'
  ),
  (
    '33333333-3333-3333-3333-333333333334',
    '11111111-1111-1111-1111-111111111111',
    'rate-limiter-v2',
    'Rate Limiter V2',
    'Token bucket rate limiter with sliding window',
    'json',
    '{"windowMs":60000,"maxRequests":100}'::jsonb,
    true,
    array['backend', 'infrastructure'],
    '{}'::text[],
    false,
    false,
    '66666666-6666-6666-6666-666666666661',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:21:49.349972+00:00',
    '[
      {"key":"default","name":"Default","value":{"windowMs":60000,"maxRequests":100}},
      {"key":"strict","name":"Strict","value":{"windowMs":60000,"maxRequests":50}}
    ]'::jsonb,
    'default'
  ),
  (
    '33333333-3333-3333-3333-333333333335',
    '11111111-1111-1111-1111-111111111111',
    'search-v3',
    'Search V3',
    'Elasticsearch-backed full-text search with typo tolerance',
    'boolean',
    'false'::jsonb,
    false,
    array['search', 'backend'],
    '{}'::text[],
    false,
    false,
    '66666666-6666-6666-6666-666666666661',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:21:49.349972+00:00',
    '[
      {"key":"on","name":"Enabled","value":true},
      {"key":"off","name":"Disabled","value":false}
    ]'::jsonb,
    'off'
  )
on conflict (id) do update
set
  name = excluded.name,
  description = excluded.description,
  enabled = excluded.enabled,
  tags = excluded.tags,
  updated_at = excluded.updated_at,
  variations = excluded.variations,
  default_variation = excluded.default_variation;

insert into public.flag_environments (
  flag_id,
  environment_id,
  enabled,
  rules,
  fallthrough,
  off_variation,
  rollout_percentage,
  kill_switch_active,
  version,
  updated_at
)
values
  ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222221', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 30, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222223', false, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 0, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333331', '22222222-2222-2222-2222-222222222221', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333331', '22222222-2222-2222-2222-222222222222', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  (
    '33333333-3333-3333-3333-333333333331',
    '22222222-2222-2222-2222-222222222223',
    true,
    '[
      {
        "id":"rule-1",
        "description":"Beta users",
        "clauses":[{"negate":false,"values":["beta","internal"],"operator":"in","attribute":"userGroup"}],
        "variation":"on",
        "rolloutPercentage":100,
        "priority":0
      }
    ]'::jsonb,
    '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb,
    '"off"'::jsonb,
    75,
    false,
    1,
    '2026-05-08T02:19:54.645629+00:00'
  ),
  ('33333333-3333-3333-3333-333333333332', '22222222-2222-2222-2222-222222222221', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333332', '22222222-2222-2222-2222-222222222222', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 50, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333332', '22222222-2222-2222-2222-222222222223', false, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 0, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333334', '22222222-2222-2222-2222-222222222221', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, 'null'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333334', '22222222-2222-2222-2222-222222222222', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, 'null'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333334', '22222222-2222-2222-2222-222222222223', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, 'null'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333335', '22222222-2222-2222-2222-222222222221', true, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 100, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333335', '22222222-2222-2222-2222-222222222222', false, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 0, false, 1, '2026-05-08T02:19:54.645629+00:00'),
  ('33333333-3333-3333-3333-333333333335', '22222222-2222-2222-2222-222222222223', false, '[]'::jsonb, '{"seed":0,"bucket_by":"key","variations":[]}'::jsonb, '"off"'::jsonb, 0, false, 1, '2026-05-08T02:19:54.645629+00:00')
on conflict (flag_id, environment_id) do update
set
  enabled = excluded.enabled,
  rules = excluded.rules,
  off_variation = excluded.off_variation,
  rollout_percentage = excluded.rollout_percentage,
  kill_switch_active = excluded.kill_switch_active,
  updated_at = excluded.updated_at;

insert into public.experiments (
  id,
  project_id,
  flag_id,
  environment_id,
  key,
  name,
  description,
  status,
  hypothesis,
  traffic_percent,
  target_metric,
  metrics,
  guardrail_metrics,
  variations,
  started_at,
  stopped_at,
  created_at,
  updated_at
)
values
  (
    '44444444-4444-4444-4444-444444444441',
    '11111111-1111-1111-1111-111111111111',
    '33333333-3333-3333-3333-333333333332',
    '22222222-2222-2222-2222-222222222222',
    'checkout-streamline',
    'Checkout Streamline Test',
    'Testing the new streamlined checkout against the legacy flow',
    'running',
    'Reducing checkout friction will improve conversion without hurting latency.',
    50,
    'checkout_conversion_rate',
    '[]'::jsonb,
    '[{"pValue":0.18,"passed":true,"message":"No statistically meaningful latency regression yet.","metricKey":"p95_latency_ms","metricName":"P95 Latency","controlValue":214,"treatmentValue":226}]'::jsonb,
    '[{"key":"control","name":"Legacy Flow","weight":50,"isControl":true},{"key":"treatment","name":"New Flow","weight":50,"isControl":false}]'::jsonb,
    '2026-04-29T02:19:54.645629+00:00',
    null,
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:19:54.645629+00:00'
  ),
  (
    '44444444-4444-4444-4444-444444444442',
    '11111111-1111-1111-1111-111111111111',
    '33333333-3333-3333-3333-333333333333',
    '22222222-2222-2222-2222-222222222222',
    'recommendation-model-v2',
    'Recommendation Model V2',
    'Comparing neural recommendations against collaborative filtering for premium users',
    'paused',
    'A neural recommender should increase CTR, but it must stay within render-time guardrails.',
    30,
    'recommendation_ctr',
    '[]'::jsonb,
    '[{"pValue":0.03,"passed":false,"message":"Treatment is degrading render time; investigate before resuming.","metricKey":"render_time_ms","metricName":"Render Time","controlValue":118,"treatmentValue":141}]'::jsonb,
    '[{"key":"control","name":"Collaborative Filtering","weight":50,"isControl":true},{"key":"treatment","name":"Neural Network","weight":50,"isControl":false}]'::jsonb,
    '2026-04-24T02:19:54.645629+00:00',
    '2026-05-06T02:19:54.645629+00:00',
    '2026-05-08T02:19:54.645629+00:00',
    '2026-05-08T02:19:54.645629+00:00'
  )
on conflict (id) do update
set
  name = excluded.name,
  description = excluded.description,
  status = excluded.status,
  guardrail_metrics = excluded.guardrail_metrics,
  variations = excluded.variations,
  updated_at = excluded.updated_at;

insert into public.experiment_results (
  experiment_id,
  status,
  total_sample_size,
  start_date,
  end_date,
  variation_results,
  guardrail_results,
  recommendation,
  last_updated
)
values
  (
    '44444444-4444-4444-4444-444444444441',
    'running',
    182430,
    '2026-04-29T02:19:54.645629+00:00',
    null,
    '[
      {"name":"Legacy Flow","isWinner":false,"isControl":true,"sampleSize":91210,"conversions":14337,"variationKey":"control","conversionRate":15.72,"credibleInterval":{"lower":15.31,"upper":16.09},"improvementOverControl":0,"probabilityToBeatControl":0.5,"isStatisticallySignificant":false},
      {"name":"New Flow","isWinner":true,"isControl":false,"sampleSize":91220,"conversions":15154,"variationKey":"treatment","conversionRate":16.61,"credibleInterval":{"lower":16.22,"upper":16.98},"improvementOverControl":5.66,"probabilityToBeatControl":0.947,"isStatisticallySignificant":true}
    ]'::jsonb,
    '[{"pValue":0.18,"passed":true,"message":"No statistically meaningful latency regression yet.","metricKey":"p95_latency_ms","metricName":"P95 Latency","controlValue":214,"treatmentValue":226}]'::jsonb,
    '{"action":"ship","reasoning":"Variant B is beating control with strong probability and no meaningful guardrail regression.","confidence":0.91,"suggestedNextSteps":["Promote to production at 100% rollout.","Continue watching latency for 24 hours after launch."]}'::jsonb,
    '2026-05-08T00:19:54.645629+00:00'
  ),
  (
    '44444444-4444-4444-4444-444444444442',
    'paused',
    54400,
    '2026-04-24T02:19:54.645629+00:00',
    '2026-05-06T02:19:54.645629+00:00',
    '[
      {"name":"Collaborative Filtering","isWinner":false,"isControl":true,"sampleSize":27120,"conversions":2532,"variationKey":"control","conversionRate":9.34,"credibleInterval":{"lower":8.98,"upper":9.68},"improvementOverControl":0,"probabilityToBeatControl":0.5,"isStatisticallySignificant":false},
      {"name":"Neural Network","isWinner":true,"isControl":false,"sampleSize":27280,"conversions":2688,"variationKey":"treatment","conversionRate":9.85,"credibleInterval":{"lower":9.49,"upper":10.18},"improvementOverControl":5.46,"probabilityToBeatControl":0.82,"isStatisticallySignificant":false}
    ]'::jsonb,
    '[{"pValue":0.03,"passed":false,"message":"Treatment is degrading render time; investigate before resuming.","metricKey":"render_time_ms","metricName":"Render Time","controlValue":118,"treatmentValue":141}]'::jsonb,
    '{"action":"iterate","reasoning":"The treatment is directionally better on CTR, but the render-time guardrail regressed enough to pause rollout.","confidence":0.74,"suggestedNextSteps":["Profile recommendation rendering.","Retry after reducing model payload size."]}'::jsonb,
    '2026-05-07T02:19:54.645629+00:00'
  )
on conflict (experiment_id) do update
set
  status = excluded.status,
  total_sample_size = excluded.total_sample_size,
  variation_results = excluded.variation_results,
  guardrail_results = excluded.guardrail_results,
  recommendation = excluded.recommendation,
  last_updated = excluded.last_updated;

insert into public.audit_logs (
  id,
  project_id,
  environment_id,
  action,
  actor_id,
  actor_email,
  resource_type,
  resource_id,
  previous_state,
  new_state,
  timestamp
)
values
  (
    '55555555-5555-5555-5555-555555555551',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222223',
    'flag.updated',
    '66666666-6666-6666-6666-666666666661',
    'demo-admin@rollout.dev',
    'flag',
    '33333333-3333-3333-3333-333333333331',
    '{"rolloutPercentage":50}'::jsonb,
    '{"rolloutPercentage":75}'::jsonb,
    '2026-05-07T23:19:54.645629+00:00'
  ),
  (
    '55555555-5555-5555-5555-555555555554',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222223',
    'targeting.updated',
    '66666666-6666-6666-6666-666666666661',
    'demo-admin@rollout.dev',
    'flag',
    '33333333-3333-3333-3333-333333333331',
    null,
    '{"rule":"beta-users"}'::jsonb,
    '2026-05-07T20:19:54.645629+00:00'
  ),
  (
    '55555555-5555-5555-5555-555555555555',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222223',
    'rollout.updated',
    '66666666-6666-6666-6666-666666666661',
    'demo-admin@rollout.dev',
    'flag',
    '33333333-3333-3333-3333-333333333332',
    '{"rolloutPercentage":0}'::jsonb,
    '{"rolloutPercentage":50}'::jsonb,
    '2026-05-07T02:19:54.645629+00:00'
  ),
  (
    '55555555-5555-5555-5555-555555555553',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'experiment.paused',
    '66666666-6666-6666-6666-666666666662',
    'pm@example.com',
    'experiment',
    '44444444-4444-4444-4444-444444444442',
    '{"status":"running"}'::jsonb,
    '{"status":"paused"}'::jsonb,
    '2026-05-06T02:19:54.645629+00:00'
  ),
  (
    '55555555-5555-5555-5555-555555555552',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'experiment.started',
    '66666666-6666-6666-6666-666666666662',
    'pm@example.com',
    'experiment',
    '44444444-4444-4444-4444-444444444441',
    null,
    '{"status":"running"}'::jsonb,
    '2026-04-29T02:19:54.645629+00:00'
  )
on conflict (id) do update
set
  action = excluded.action,
  previous_state = excluded.previous_state,
  new_state = excluded.new_state,
  timestamp = excluded.timestamp;
