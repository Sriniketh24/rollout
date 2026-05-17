import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "npm:@supabase/supabase-js@2";

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers":
    "authorization, x-client-info, apikey, content-type, x-rollout-guest-token",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
  "Content-Type": "application/json",
};

type Json = null | boolean | number | string | Json[] | { [key: string]: Json };
type Supabase = ReturnType<typeof createClient>;

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: corsHeaders });
}

async function sha256(input: string) {
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(input),
  );
  return Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}

function normalizeVariationValue(
  value: Json,
  variations: Array<{ key: string; value: Json }>,
): string {
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

function buildChanges(
  previousState: Record<string, Json> | null,
  newState: Record<string, Json> | null,
) {
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

function parseVariationValue(type: string, rawValue: unknown): Json {
  if (typeof rawValue !== "string") return rawValue as Json;
  if (type === "boolean") return rawValue === "true";
  if (type === "number") return Number(rawValue);
  if (type === "json") {
    try {
      return JSON.parse(rawValue) as Json;
    } catch {
      return rawValue;
    }
  }
  return rawValue;
}

async function findProjectForGuestToken(supabase: Supabase, token: string) {
  const tokenHash = await sha256(token);
  const { data, error } = await supabase
    .from("guest_workspace_tokens")
    .select("project_id")
    .eq("token_hash", tokenHash)
    .single();

  if (error || !data) {
    throw new Error("invalid_guest_token");
  }

  await supabase
    .from("guest_workspace_tokens")
    .update({ last_used_at: new Date().toISOString() })
    .eq("token_hash", tokenHash);

  return data.project_id as string;
}

async function loadDatasetByProjectId(supabase: Supabase, projectId: string) {
  const { data: project, error: projectError } = await supabase
    .from("projects")
    .select("*")
    .eq("id", projectId)
    .single();

  if (projectError || !project) {
    throw new Error(projectError?.message ?? "project_not_found");
  }

  return loadDataset(supabase, project);
}

async function loadDatasetByProjectKey(supabase: Supabase, projectKey: string) {
  const { data: project, error: projectError } = await supabase
    .from("projects")
    .select("*")
    .eq("key", projectKey)
    .single();

  if (projectError || !project) {
    throw new Error(projectError?.message ?? "project_not_found");
  }

  return loadDataset(supabase, project);
}

async function loadDataset(supabase: Supabase, project: Record<string, Json>) {
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
    supabase
      .from("environments")
      .select("*")
      .eq("project_id", project.id)
      .order("sort_order"),
    supabase
      .from("flags")
      .select("*")
      .eq("project_id", project.id)
      .order("created_at", { ascending: true }),
    supabase
      .from("flag_environments")
      .select("*, environments!inner(project_id)")
      .eq("environments.project_id", project.id),
    supabase
      .from("experiments")
      .select("*")
      .eq("project_id", project.id)
      .order("created_at", { ascending: false }),
    supabase.from("experiment_results").select("*"),
    supabase
      .from("audit_logs")
      .select("*")
      .eq("project_id", project.id)
      .order("timestamp", { ascending: false }),
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
    throw new Error(firstError.message);
  }

  const users = usersResult.data ?? [];
  const environments = environmentsResult.data ?? [];
  const flags = flagsResult.data ?? [];
  const flagEnvironments = flagEnvironmentsResult.data ?? [];
  const experiments = experimentsResult.data ?? [];
  const experimentResults = experimentResultsResult.data ?? [];
  const auditLogs = auditLogsResult.data ?? [];

  const usersById = new Map(users.map((user) => [user.id, user]));
  const environmentsById = new Map(
    environments.map((environment) => [environment.id, environment]),
  );
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
        const rules = Array.isArray(environmentConfig.rules)
          ? environmentConfig.rules
          : [];
        acc[environmentConfig.environment_id] = {
          environmentId: environmentConfig.environment_id,
          enabled: environmentConfig.enabled,
          rolloutPercentage: environmentConfig.rollout_percentage,
          targetingRules: rules.map((rule: Record<string, Json>) => ({
            id: String(rule.id ?? crypto.randomUUID()),
            description: String(rule.description ?? ""),
            clauses: Array.isArray(rule.clauses) ? rule.clauses : [],
            variation: normalizeVariationValue(
              (rule.variation ?? null) as Json,
              variations,
            ),
            rolloutPercentage: typeof rule.rolloutPercentage === "number"
              ? rule.rolloutPercentage
              : typeof rule.rollout_percentage === "number"
              ? rule.rollout_percentage
              : 100,
            priority: typeof rule.priority === "number" ? rule.priority : 0,
          })),
          offVariation: normalizeVariationValue(
            environmentConfig.off_variation as Json,
            variations,
          ),
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
      createdBy: usersById.get(flag.created_by)?.name ??
        flag.created_by ??
        "Unknown",
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
    variations: Array.isArray(experiment.variations)
      ? experiment.variations
      : [],
    trafficPercentage: experiment.traffic_percent,
    startDate: experiment.started_at ?? undefined,
    endDate: experiment.stopped_at ?? undefined,
    targetMetric: experiment.target_metric,
    guardrailMetrics: Array.isArray(experiment.guardrail_metrics)
      ? experiment.guardrail_metrics
      : [],
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
        variationResults: Array.isArray(result.variation_results)
          ? result.variation_results
          : [],
        guardrailResults: Array.isArray(result.guardrail_results)
          ? result.guardrail_results
          : [],
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
      ? {
        environment: environmentsById.get(entry.environment_id)?.key ??
          entry.environment_id,
      }
      : undefined,
    projectId: entry.project_id,
    environmentId: entry.environment_id ?? undefined,
  }));

  const owner = users.find((user) => user.role === "owner") ?? users[0];

  return {
    project: {
      id: project.id,
      key: project.key,
      name: project.name,
      description: String(project.key).startsWith("guest-")
        ? "Editable guest workspace"
        : "Supabase-backed demo project for the Rollout control plane",
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
  };
}

async function createGuestWorkspace(supabase: Supabase) {
  const token = `guest_${crypto.randomUUID()}_${crypto.randomUUID()}`;
  const tokenHash = await sha256(token);
  const guestId = crypto.randomUUID();
  const projectKey = `guest-${guestId.slice(0, 12)}`;
  const guestUserId = crypto.randomUUID();
  const now = new Date().toISOString();

  const { data: user, error: userError } = await supabase
    .from("users")
    .insert({
      id: guestUserId,
      email: `${projectKey}@rollout.dev`,
      name: "Guest User",
      role: "owner",
      api_key: `rol_${guestId.replaceAll("-", "")}`,
      created_at: now,
    })
    .select("*")
    .single();
  if (userError) throw userError;

  const { data: project, error: projectError } = await supabase
    .from("projects")
    .insert({
      key: projectKey,
      name: "Guest Workspace",
      created_at: now,
      updated_at: now,
    })
    .select("*")
    .single();
  if (projectError) throw projectError;

  const { error: membershipError } = await supabase.from("user_projects").insert({
    user_id: user.id,
    project_id: project.id,
    role: "owner",
    created_at: now,
  });
  if (membershipError) throw membershipError;

  const { error: tokenError } = await supabase
    .from("guest_workspace_tokens")
    .insert({ token_hash: tokenHash, project_id: project.id });
  if (tokenError) throw tokenError;

  const { error: envError } = await supabase.from("environments").insert([
    {
      project_id: project.id,
      key: "development",
      name: "Development",
      color: "#3b82f6",
      description: "Local development environment",
      production: false,
      sort_order: 0,
    },
    {
      project_id: project.id,
      key: "staging",
      name: "Staging",
      color: "#eab308",
      description: "Pre-production testing",
      production: false,
      sort_order: 1,
    },
    {
      project_id: project.id,
      key: "production",
      name: "Production",
      color: "#ef4444",
      description: "Live production environment",
      production: true,
      sort_order: 2,
    },
  ]);
  if (envError) throw envError;

  await supabase.from("audit_logs").insert({
    project_id: project.id,
    action: "project.updated",
    actor_id: user.id,
    actor_email: user.email,
    resource_type: "project",
    resource_id: project.id,
    previous_state: null,
    new_state: { name: project.name },
  });

  return { token, dataset: await loadDataset(supabase, project) };
}

async function createFlag(
  supabase: Supabase,
  projectId: string,
  payload: Record<string, Json>,
) {
  const { data: owner } = await supabase
    .from("user_projects")
    .select("users(*)")
    .eq("project_id", projectId)
    .limit(1)
    .single();

  const actor = Array.isArray(owner?.users) ? owner?.users[0] : owner?.users;
  const variations = Array.isArray(payload.variations)
    ? payload.variations.map((variation) => ({
      ...(variation as Record<string, Json>),
      value: parseVariationValue(
        String(payload.type ?? "boolean"),
        (variation as Record<string, Json>).value,
      ),
    }))
    : [];
  const defaultVariation = String(payload.defaultVariation ?? "");
  const defaultValue =
    variations.find((variation) => variation.key === defaultVariation)?.value ??
      null;

  const { data: flag, error: flagError } = await supabase
    .from("flags")
    .insert({
      project_id: projectId,
      key: payload.key,
      name: payload.name,
      description: payload.description ?? "",
      type: payload.type ?? "boolean",
      default_value: defaultValue,
      enabled: false,
      tags: Array.isArray(payload.tags) ? payload.tags : [],
      created_by: actor?.id,
      variations,
      default_variation: defaultVariation,
    })
    .select("*")
    .single();
  if (flagError) throw flagError;

  const { data: environments, error: envLoadError } = await supabase
    .from("environments")
    .select("id")
    .eq("project_id", projectId);
  if (envLoadError) throw envLoadError;

  if (environments?.length) {
    const { error: flagEnvError } = await supabase
      .from("flag_environments")
      .insert(
        environments.map((environment) => ({
          flag_id: flag.id,
          environment_id: environment.id,
          enabled: false,
          rules: [],
          off_variation: defaultVariation,
          rollout_percentage: 0,
          kill_switch_active: false,
        })),
      );
    if (flagEnvError) throw flagEnvError;
  }

  await supabase.from("audit_logs").insert({
    project_id: projectId,
    action: "flag.created",
    actor_id: actor?.id,
    actor_email: actor?.email ?? "guest@rollout.dev",
    resource_type: "flag",
    resource_id: flag.id,
    previous_state: null,
    new_state: { key: payload.key, name: payload.name },
  });
}

async function updateFlagEnvironment(
  supabase: Supabase,
  projectId: string,
  payload: Record<string, Json>,
) {
  const flagId = String(payload.flagId ?? "");
  const environmentId = String(payload.environmentId ?? "");

  const { data: previous } = await supabase
    .from("flag_environments")
    .select("*")
    .eq("flag_id", flagId)
    .eq("environment_id", environmentId)
    .single();

  const { data: owner } = await supabase
    .from("user_projects")
    .select("users(*)")
    .eq("project_id", projectId)
    .limit(1)
    .single();
  const actor = Array.isArray(owner?.users) ? owner?.users[0] : owner?.users;

  const nextState = {
    enabled: Boolean(payload.enabled),
    rolloutPercentage: Number(payload.rolloutPercentage ?? 0),
    targetingRules: Array.isArray(payload.targetingRules)
      ? payload.targetingRules
      : [],
    killSwitchActive: Boolean(payload.killSwitchActive),
  };

  const { error: updateError } = await supabase
    .from("flag_environments")
    .update({
      enabled: nextState.enabled,
      rollout_percentage: nextState.rolloutPercentage,
      rules: nextState.targetingRules,
      kill_switch_active: nextState.killSwitchActive,
      updated_at: new Date().toISOString(),
    })
    .eq("flag_id", flagId)
    .eq("environment_id", environmentId);
  if (updateError) throw updateError;

  await supabase.from("audit_logs").insert({
    project_id: projectId,
    environment_id: environmentId,
    action: "flag.updated",
    actor_id: actor?.id,
    actor_email: actor?.email ?? "guest@rollout.dev",
    resource_type: "flag",
    resource_id: flagId,
    previous_state: previous
      ? {
        enabled: previous.enabled,
        rolloutPercentage: previous.rollout_percentage,
        targetingRules: previous.rules,
        killSwitchActive: previous.kill_switch_active,
      }
      : null,
    new_state: nextState,
  });
}

async function createEnvironment(
  supabase: Supabase,
  projectId: string,
  payload: Record<string, Json>,
) {
  const { data: owner } = await supabase
    .from("user_projects")
    .select("users(*)")
    .eq("project_id", projectId)
    .limit(1)
    .single();
  const actor = Array.isArray(owner?.users) ? owner?.users[0] : owner?.users;

  const { data: existingEnvironments } = await supabase
    .from("environments")
    .select("id")
    .eq("project_id", projectId);

  const { data: environment, error: environmentError } = await supabase
    .from("environments")
    .insert({
      project_id: projectId,
      key: payload.key,
      name: payload.name,
      color: payload.color ?? "#3b82f6",
      description: payload.description ?? "",
      production: Boolean(payload.production),
      sort_order: existingEnvironments?.length ?? 0,
    })
    .select("*")
    .single();
  if (environmentError) throw environmentError;

  const { data: flags, error: flagsError } = await supabase
    .from("flags")
    .select("id, default_variation")
    .eq("project_id", projectId);
  if (flagsError) throw flagsError;

  if (flags?.length) {
    const { error: flagEnvError } = await supabase
      .from("flag_environments")
      .insert(
        flags.map((flag) => ({
          flag_id: flag.id,
          environment_id: environment.id,
          enabled: false,
          rules: [],
          off_variation: flag.default_variation,
          rollout_percentage: 0,
          kill_switch_active: false,
        })),
      );
    if (flagEnvError) throw flagEnvError;
  }

  await supabase.from("audit_logs").insert({
    project_id: projectId,
    environment_id: environment.id,
    action: "environment.created",
    actor_id: actor?.id,
    actor_email: actor?.email ?? "guest@rollout.dev",
    resource_type: "environment",
    resource_id: environment.id,
    previous_state: null,
    new_state: { key: payload.key, name: payload.name },
  });
}

Deno.serve(async (req) => {
  if (req.method === "OPTIONS") {
    return new Response("ok", { headers: corsHeaders });
  }

  const url = Deno.env.get("SUPABASE_URL");
  const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY");

  if (!url || !serviceRoleKey) {
    return jsonResponse({ error: "missing_supabase_env" }, 500);
  }

  const supabase = createClient(url, serviceRoleKey, {
    auth: { autoRefreshToken: false, persistSession: false },
  });

  try {
    if (req.method === "GET") {
      const requestUrl = new URL(req.url);
      const guestToken = req.headers.get("x-rollout-guest-token");
      if (guestToken) {
        const projectId = await findProjectForGuestToken(supabase, guestToken);
        return jsonResponse(await loadDatasetByProjectId(supabase, projectId));
      }

      const projectKey = requestUrl.searchParams.get("projectKey") ??
        "rollout-demo";
      return jsonResponse(await loadDatasetByProjectKey(supabase, projectKey));
    }

    if (req.method !== "POST") {
      return jsonResponse({ error: "method_not_allowed" }, 405);
    }

    const body = await req.json().catch(() => ({}));
    const action = String(body.action ?? "");

    if (action === "create_guest_workspace") {
      return jsonResponse(await createGuestWorkspace(supabase), 201);
    }

    const token = String(body.token ?? "");
    if (!token) {
      return jsonResponse({ error: "missing_guest_token" }, 401);
    }

    const projectId = await findProjectForGuestToken(supabase, token);
    const payload = (body.payload ?? {}) as Record<string, Json>;

    if (action === "create_flag") {
      await createFlag(supabase, projectId, payload);
    } else if (action === "update_flag_environment") {
      await updateFlagEnvironment(supabase, projectId, payload);
    } else if (action === "create_environment") {
      await createEnvironment(supabase, projectId, payload);
    } else {
      return jsonResponse({ error: "unknown_action" }, 400);
    }

    return jsonResponse({ dataset: await loadDatasetByProjectId(supabase, projectId) });
  } catch (error) {
    const message = error instanceof Error ? error.message : "request_failed";
    const status = message === "invalid_guest_token" ? 401 : 500;
    return jsonResponse({ error: message }, status);
  }
});
