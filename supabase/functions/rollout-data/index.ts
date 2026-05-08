import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "npm:@supabase/supabase-js@2";

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
  "Access-Control-Allow-Methods": "GET, OPTIONS",
  "Content-Type": "application/json",
};

type Json = null | boolean | number | string | Json[] | { [key: string]: Json };

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: corsHeaders });
}

function normalizeVariationValue(value: Json, variations: Array<{ key: string; value: Json }>): string {
  for (const variation of variations) {
    if (JSON.stringify(variation.value) === JSON.stringify(value)) {
      return variation.key;
    }
  }

  if (typeof value === "string") {
    const found = variations.find((variation) => variation.key === value);
    if (found) return found.key;
  }

  return String(value ?? "");
}

function buildChanges(previousState: Record<string, Json> | null, newState: Record<string, Json> | null) {
  if (!previousState && !newState) return undefined;

  const keys = new Set([
    ...Object.keys(previousState ?? {}),
    ...Object.keys(newState ?? {}),
  ]);

  const changes: Record<string, { before: Json; after: Json }> = {};
  for (const key of keys) {
    changes[key] = {
      before: previousState?.[key] ?? null,
      after: newState?.[key] ?? null,
    };
  }

  return changes;
}

Deno.serve(async (req) => {
  if (req.method === "OPTIONS") {
    return new Response("ok", { headers: corsHeaders });
  }

  if (req.method !== "GET") {
    return jsonResponse({ error: "method_not_allowed" }, 405);
  }

  const url = Deno.env.get("SUPABASE_URL");
  const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY");

  if (!url || !serviceRoleKey) {
    return jsonResponse({ error: "missing_supabase_env" }, 500);
  }

  const projectKey = new URL(req.url).searchParams.get("projectKey") ?? "rollout-demo";
  const supabase = createClient(url, serviceRoleKey, {
    auth: { autoRefreshToken: false, persistSession: false },
  });

  const { data: project, error: projectError } = await supabase
    .from("projects")
    .select("*")
    .eq("key", projectKey)
    .single();

  if (projectError || !project) {
    return jsonResponse({ error: "project_not_found", details: projectError?.message }, 404);
  }

  const [
    usersResult,
    environmentsResult,
    flagsResult,
    flagEnvironmentsResult,
    experimentsResult,
    experimentResultsResult,
    auditLogsResult,
  ] = await Promise.all([
    supabase.from("users").select("*"),
    supabase.from("environments").select("*").eq("project_id", project.id).order("sort_order"),
    supabase.from("flags").select("*").eq("project_id", project.id).order("created_at", { ascending: true }),
    supabase
      .from("flag_environments")
      .select("*, environments!inner(project_id)")
      .eq("environments.project_id", project.id),
    supabase.from("experiments").select("*").eq("project_id", project.id).order("created_at", { ascending: false }),
    supabase.from("experiment_results").select("*"),
    supabase.from("audit_logs").select("*").eq("project_id", project.id).order("timestamp", { ascending: false }),
  ]);

  const firstError = [
    usersResult.error,
    environmentsResult.error,
    flagsResult.error,
    flagEnvironmentsResult.error,
    experimentsResult.error,
    experimentResultsResult.error,
    auditLogsResult.error,
  ].find(Boolean);

  if (firstError) {
    return jsonResponse({ error: "query_failed", details: firstError?.message }, 500);
  }

  const users = usersResult.data ?? [];
  const environments = environmentsResult.data ?? [];
  const flags = flagsResult.data ?? [];
  const flagEnvironments = flagEnvironmentsResult.data ?? [];
  const experiments = experimentsResult.data ?? [];
  const experimentResults = experimentResultsResult.data ?? [];
  const auditLogs = auditLogsResult.data ?? [];

  const usersById = new Map(users.map((user) => [user.id, user]));
  const environmentsById = new Map(environments.map((environment) => [environment.id, environment]));
  const flagsById = new Map(flags.map((flag) => [flag.id, flag]));

  const normalizedEnvironments = environments.map((environment) => ({
    id: environment.id,
    key: environment.key,
    name: environment.name,
    color: environment.color,
    description: environment.description,
    production: environment.production,
    createdAt: environment.created_at,
    updatedAt: environment.updated_at,
    projectId: environment.project_id,
    order: environment.sort_order,
  }));

  const normalizedFlags = flags.map((flag) => {
    const variations = Array.isArray(flag.variations) ? flag.variations : [];
    const envConfigs = flagEnvironments
      .filter((environmentConfig) => environmentConfig.flag_id === flag.id)
      .reduce<Record<string, unknown>>((acc, environmentConfig) => {
        const rules = Array.isArray(environmentConfig.rules) ? environmentConfig.rules : [];
        acc[environmentConfig.environment_id] = {
          environmentId: environmentConfig.environment_id,
          enabled: environmentConfig.enabled,
          rolloutPercentage: environmentConfig.rollout_percentage,
          targetingRules: rules.map((rule: Record<string, Json>) => ({
            id: String(rule.id ?? crypto.randomUUID()),
            description: String(rule.description ?? ""),
            clauses: Array.isArray(rule.clauses) ? rule.clauses : [],
            variation: normalizeVariationValue((rule.variation ?? null) as Json, variations),
            rolloutPercentage: typeof rule.rolloutPercentage === "number"
              ? rule.rolloutPercentage
              : typeof rule.rollout_percentage === "number"
              ? rule.rollout_percentage
              : 100,
            priority: typeof rule.priority === "number" ? rule.priority : 0,
          })),
          offVariation: normalizeVariationValue(environmentConfig.off_variation as Json, variations),
          killSwitchActive: environmentConfig.kill_switch_active,
          lastModified: environmentConfig.updated_at,
        };
        return acc;
      }, {});

    return {
      id: flag.id,
      key: flag.key,
      name: flag.name,
      description: flag.description,
      type: flag.type,
      tags: flag.tags ?? [],
      variations,
      defaultVariation: flag.default_variation,
      environments: envConfigs,
      createdAt: flag.created_at,
      updatedAt: flag.updated_at,
      createdBy: usersById.get(flag.created_by)?.name ?? flag.created_by ?? "Unknown",
      archived: flag.archived,
      projectId: flag.project_id,
    };
  });

  const flagKeyById = new Map(flags.map((flag) => [flag.id, flag.key]));

  const normalizedExperiments = experiments.map((experiment) => ({
    id: experiment.id,
    key: experiment.key,
    name: experiment.name,
    description: experiment.description,
    flagId: experiment.flag_id,
    flagKey: flagKeyById.get(experiment.flag_id),
    status: experiment.status,
    variations: Array.isArray(experiment.variations) ? experiment.variations : [],
    trafficPercentage: experiment.traffic_percent,
    startDate: experiment.started_at ?? undefined,
    endDate: experiment.stopped_at ?? undefined,
    targetMetric: experiment.target_metric,
    guardrailMetrics: Array.isArray(experiment.guardrail_metrics) ? experiment.guardrail_metrics : [],
    createdAt: experiment.created_at,
    updatedAt: experiment.updated_at,
    projectId: experiment.project_id,
  }));

  const experimentResultsById = Object.fromEntries(
    experimentResults.map((result) => [
      result.experiment_id,
      {
        experimentId: result.experiment_id,
        status: result.status,
        totalSampleSize: result.total_sample_size,
        startDate: result.start_date,
        endDate: result.end_date ?? undefined,
        variationResults: Array.isArray(result.variation_results) ? result.variation_results : [],
        guardrailResults: Array.isArray(result.guardrail_results) ? result.guardrail_results : [],
        recommendation: result.recommendation ?? undefined,
        lastUpdated: result.last_updated,
      },
    ]),
  );

  const normalizedAuditEntries = auditLogs.map((entry) => ({
    id: entry.id,
    action: entry.action,
    actor: usersById.get(entry.actor_id)?.name ?? entry.actor_email,
    actorEmail: entry.actor_email,
    targetType: entry.resource_type,
    targetId: entry.resource_id,
    targetName: flagsById.get(entry.resource_id)?.name ?? undefined,
    timestamp: entry.timestamp,
    changes: buildChanges(entry.previous_state, entry.new_state),
    metadata: entry.environment_id
      ? { environment: environmentsById.get(entry.environment_id)?.key ?? entry.environment_id }
      : undefined,
    projectId: entry.project_id,
    environmentId: entry.environment_id ?? undefined,
  }));

  const owner = users.find((user) => user.role === "owner") ?? users[0];

  return jsonResponse({
    project: {
      id: project.id,
      key: project.key,
      name: project.name,
      description: "Supabase-backed demo project for the Rollout control plane",
      apiKey: owner?.api_key ?? "",
      serverApiKey: owner?.api_key ?? "",
      createdAt: project.created_at,
      updatedAt: project.updated_at,
    },
    environments: normalizedEnvironments,
    flags: normalizedFlags,
    experiments: normalizedExperiments,
    experimentResultsById,
    auditEntries: normalizedAuditEntries,
  });
});
