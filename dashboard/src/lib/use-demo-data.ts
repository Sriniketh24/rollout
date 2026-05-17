"use client";

import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
import { fallbackDemoDataset, type DemoDataset } from "@/lib/demo-dataset";
import { supabase } from "@/lib/supabase";
import { useAuth } from "@/lib/auth-context";
import type {
  AuditEntry,
  AuditAction,
  Clause,
  Environment,
  EnvironmentConfig,
  Experiment,
  ExperimentResults,
  Flag,
  FlagType,
  FlagVariation,
  GuardrailResult,
  GuardrailMetric,
  TargetingRule,
  VariationResult,
} from "@/lib/types";

type JsonRecord = Record<string, unknown>;

export interface CreateFlagInput {
  key: string;
  name: string;
  description: string;
  type: FlagType;
  tags: string[];
  variations: FlagVariation[];
  defaultVariation: string;
}

export interface UpdateFlagEnvironmentInput {
  flagId: string;
  environmentId: string;
  enabled: boolean;
  rolloutPercentage: number;
  targetingRules: TargetingRule[];
  killSwitchActive: boolean;
}

export interface CreateEnvironmentInput {
  key: string;
  name: string;
  color: string;
  description: string;
  production: boolean;
}

function toArray<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

function normalizeVariationValue(
  value: unknown,
  variations: Array<{ key: string; value: unknown }>
) {
  for (const variation of variations) {
    if (JSON.stringify(variation.value) === JSON.stringify(value)) {
      return variation.key;
    }
  }

  if (typeof value === "string" && variations.some((item) => item.key === value)) {
    return value;
  }

  return String(value ?? "");
}

function buildChanges(
  previousState: JsonRecord | null,
  newState: JsonRecord | null
): AuditEntry["changes"] {
  if (!previousState && !newState) return undefined;

  const keys = new Set([
    ...Object.keys(previousState ?? {}),
    ...Object.keys(newState ?? {}),
  ]);

  return Object.fromEntries(
    Array.from(keys).map((key) => [
      key,
      {
        before: previousState?.[key] ?? null,
        after: newState?.[key] ?? null,
      },
    ])
  );
}

function parseVariationValue(type: FlagType, rawValue: unknown) {
  if (typeof rawValue !== "string") return rawValue;

  if (type === "boolean") return rawValue === "true";
  if (type === "number") return Number(rawValue);
  if (type === "json") {
    try {
      return JSON.parse(rawValue);
    } catch {
      return rawValue;
    }
  }

  return rawValue;
}

function normalizeDataset(rows: {
  project: JsonRecord;
  userRows: JsonRecord[];
  environments: JsonRecord[];
  flags: JsonRecord[];
  flagEnvironments: JsonRecord[];
  experiments: JsonRecord[];
  experimentResults: JsonRecord[];
  auditLogs: JsonRecord[];
}): DemoDataset {
  const usersById = new Map(rows.userRows.map((user) => [user.id, user]));
  const environmentsById = new Map(
    rows.environments.map((environment) => [environment.id, environment])
  );
  const flagsById = new Map(rows.flags.map((flag) => [flag.id, flag]));

  const environments: Environment[] = rows.environments.map((environment) => ({
    id: String(environment.id),
    key: String(environment.key),
    name: String(environment.name),
    color: String(environment.color),
    description: String(environment.description ?? ""),
    production: Boolean(environment.production),
    createdAt: String(environment.created_at),
    updatedAt: String(environment.updated_at),
    projectId: String(environment.project_id),
    order: Number(environment.sort_order ?? 0),
  }));

  const flags: Flag[] = rows.flags.map((flag) => {
    const variations = toArray<FlagVariation>(flag.variations);
    const environmentsForFlag = rows.flagEnvironments
      .filter((entry) => entry.flag_id === flag.id)
      .reduce<Record<string, EnvironmentConfig>>((acc, entry) => {
        const rules = toArray<JsonRecord>(entry.rules).map((rule) => ({
          id: String(rule.id ?? crypto.randomUUID()),
          description: String(rule.description ?? ""),
          clauses: toArray<Clause>(rule.clauses),
          variation: normalizeVariationValue(rule.variation, variations),
          rolloutPercentage: Number(
            rule.rolloutPercentage ?? rule.rollout_percentage ?? 100
          ),
          priority: Number(rule.priority ?? 0),
        }));

        acc[String(entry.environment_id)] = {
          environmentId: String(entry.environment_id),
          enabled: Boolean(entry.enabled),
          rolloutPercentage: Number(entry.rollout_percentage ?? 0),
          targetingRules: rules,
          offVariation: normalizeVariationValue(entry.off_variation, variations),
          killSwitchActive: Boolean(entry.kill_switch_active),
          lastModified: String(entry.updated_at),
        };
        return acc;
      }, {});

    const createdBy = usersById.get(flag.created_by);
    return {
      id: String(flag.id),
      key: String(flag.key),
      name: String(flag.name),
      description: String(flag.description ?? ""),
      type: String(flag.type) as FlagType,
      tags: toArray<string>(flag.tags),
      variations,
      defaultVariation: String(flag.default_variation ?? ""),
      environments: environmentsForFlag,
      createdAt: String(flag.created_at),
      updatedAt: String(flag.updated_at),
      createdBy: String(createdBy?.name ?? "Workspace user"),
      archived: Boolean(flag.archived),
      projectId: String(flag.project_id),
    };
  });

  const flagKeyById = new Map(rows.flags.map((flag) => [flag.id, flag.key]));
  const experiments: Experiment[] = rows.experiments.map((experiment) => ({
    id: String(experiment.id),
    key: String(experiment.key),
    name: String(experiment.name),
    description: String(experiment.description ?? ""),
    flagId: String(experiment.flag_id),
    flagKey: String(flagKeyById.get(experiment.flag_id) ?? ""),
    status: String(experiment.status) as Experiment["status"],
    variations: toArray(experiment.variations),
    trafficPercentage: Number(experiment.traffic_percent ?? 0),
    startDate: experiment.started_at ? String(experiment.started_at) : undefined,
    endDate: experiment.stopped_at ? String(experiment.stopped_at) : undefined,
    targetMetric: String(experiment.target_metric ?? "conversion_rate"),
    guardrailMetrics: toArray<GuardrailMetric>(experiment.guardrail_metrics),
    createdAt: String(experiment.created_at),
    updatedAt: String(experiment.updated_at),
    projectId: String(experiment.project_id),
  }));

  const experimentResultsById = Object.fromEntries(
    rows.experimentResults.map((result) => [
      String(result.experiment_id),
      {
        experimentId: String(result.experiment_id),
        status: String(result.status) as ExperimentResults["status"],
        totalSampleSize: Number(result.total_sample_size ?? 0),
        startDate: String(result.start_date),
        endDate: result.end_date ? String(result.end_date) : undefined,
        variationResults: toArray<VariationResult>(result.variation_results),
        guardrailResults: toArray<GuardrailResult>(result.guardrail_results),
        recommendation: (result.recommendation as ExperimentResults["recommendation"]) ?? undefined,
        lastUpdated: String(result.last_updated),
      },
    ])
  );

  const auditEntries: AuditEntry[] = rows.auditLogs.map((entry) => ({
    id: String(entry.id),
    action: String(entry.action) as AuditAction,
    actor: String(usersById.get(entry.actor_id)?.name ?? entry.actor_email ?? "User"),
    actorEmail: String(entry.actor_email ?? ""),
    targetType: String(entry.resource_type),
    targetId: String(entry.resource_id),
    targetName: String(flagsById.get(entry.resource_id)?.name ?? ""),
    timestamp: String(entry.timestamp),
    changes: buildChanges(
      (entry.previous_state as JsonRecord | null) ?? null,
      (entry.new_state as JsonRecord | null) ?? null
    ),
    metadata: entry.environment_id
      ? { environment: environmentsById.get(entry.environment_id)?.key ?? entry.environment_id }
      : undefined,
    projectId: String(entry.project_id),
    environmentId: entry.environment_id ? String(entry.environment_id) : undefined,
  }));

  const currentUser = rows.userRows[0];

  return {
    project: {
      id: String(rows.project.id),
      key: String(rows.project.key),
      name: String(rows.project.name),
      description: "Authenticated workspace",
      apiKey: String(currentUser?.api_key ?? ""),
      serverApiKey: String(currentUser?.api_key ?? ""),
      createdAt: String(rows.project.created_at),
      updatedAt: String(rows.project.updated_at),
    },
    environments,
    flags,
    experiments,
    experimentResultsById,
    auditEntries,
  };
}

export function useDemoData() {
  const { session, user, isLoading: authLoading } = useAuth();
  const [data, setData] = useState<DemoDataset>(fallbackDemoDataset);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const isAuthenticated = Boolean(session && user);

  const loadDemo = useCallback(async () => {
    const response = await fetch("/api/demo-data", { cache: "no-store" });
    if (!response.ok) throw new Error("Unable to load public demo data");
    return (await response.json()) as DemoDataset;
  }, []);

  const loadWorkspace = useCallback(async () => {
    const { data: projectId, error: workspaceError } = await supabase.rpc(
      "create_default_workspace",
      { display_name: user?.user_metadata?.name ?? null }
    );
    if (workspaceError) throw workspaceError;

    const projectQuery = await supabase
      .from("projects")
      .select("*")
      .eq("id", projectId)
      .single();
    if (projectQuery.error) throw projectQuery.error;

    const [
      userRows,
      environments,
      flags,
      flagEnvironments,
      experiments,
      experimentResults,
      auditLogs,
    ] = await Promise.all([
      supabase.from("users").select("*"),
      supabase
        .from("environments")
        .select("*")
        .eq("project_id", projectId)
        .order("sort_order"),
      supabase
        .from("flags")
        .select("*")
        .eq("project_id", projectId)
        .order("created_at", { ascending: true }),
      supabase.from("flag_environments").select("*"),
      supabase
        .from("experiments")
        .select("*")
        .eq("project_id", projectId)
        .order("created_at", { ascending: false }),
      supabase.from("experiment_results").select("*"),
      supabase
        .from("audit_logs")
        .select("*")
        .eq("project_id", projectId)
        .order("timestamp", { ascending: false }),
    ]);

    const firstError = [
      userRows.error,
      environments.error,
      flags.error,
      flagEnvironments.error,
      experiments.error,
      experimentResults.error,
      auditLogs.error,
    ].find(Boolean);
    if (firstError) throw firstError;

    return normalizeDataset({
      project: projectQuery.data,
      userRows: userRows.data ?? [],
      environments: environments.data ?? [],
      flags: flags.data ?? [],
      flagEnvironments: flagEnvironments.data ?? [],
      experiments: experiments.data ?? [],
      experimentResults: experimentResults.data ?? [],
      auditLogs: auditLogs.data ?? [],
    });
  }, [user?.user_metadata?.name]);

  const reload = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const nextData = isAuthenticated ? await loadWorkspace() : await loadDemo();
      startTransition(() => {
        setData(nextData);
      });
    } catch (loadError) {
      setError(
        loadError instanceof Error ? loadError.message : "Unable to load workspace"
      );
      if (!isAuthenticated) {
        setData(fallbackDemoDataset);
      }
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated, loadDemo, loadWorkspace]);

  useEffect(() => {
    if (authLoading) return;
    const timeout = window.setTimeout(() => void reload(), 0);
    return () => window.clearTimeout(timeout);
  }, [authLoading, reload]);

  const currentUserRow = useMemo(
    () => ({
      id: user?.id ?? "",
      email: user?.email ?? "",
    }),
    [user?.email, user?.id]
  );

  async function createFlag(input: CreateFlagInput) {
    if (!isAuthenticated) {
      throw new Error("Sign in to create persistent flags.");
    }

    const now = new Date().toISOString();
    const parsedVariations = input.variations.map((variation) => ({
      ...variation,
      value: parseVariationValue(input.type, variation.value),
    }));
    const defaultValue =
      parsedVariations.find((variation) => variation.key === input.defaultVariation)
        ?.value ?? null;

    const { data: insertedFlag, error: flagError } = await supabase
      .from("flags")
      .insert({
        project_id: data.project.id,
        key: input.key,
        name: input.name,
        description: input.description,
        type: input.type,
        default_value: defaultValue,
        enabled: false,
        tags: input.tags,
        created_by: currentUserRow.id,
        variations: parsedVariations,
        default_variation: input.defaultVariation,
      })
      .select("*")
      .single();
    if (flagError) throw flagError;

    const envRows = data.environments.map((environment) => ({
      flag_id: insertedFlag.id,
      environment_id: environment.id,
      enabled: false,
      rules: [],
      off_variation: input.defaultVariation,
      rollout_percentage: 0,
      kill_switch_active: false,
      updated_at: now,
    }));
    const { error: envError } = await supabase
      .from("flag_environments")
      .insert(envRows);
    if (envError) throw envError;

    await supabase.from("audit_logs").insert({
      project_id: data.project.id,
      action: "flag.created",
      actor_id: currentUserRow.id,
      actor_email: currentUserRow.email,
      resource_type: "flag",
      resource_id: insertedFlag.id,
      previous_state: null,
      new_state: { key: input.key, name: input.name },
    });

    await reload();
    return insertedFlag.id as string;
  }

  async function updateFlagEnvironment(input: UpdateFlagEnvironmentInput) {
    if (!isAuthenticated) {
      throw new Error("Sign in to edit persistent flags.");
    }

    const flag = data.flags.find((entry) => entry.id === input.flagId);
    const previous = flag?.environments[input.environmentId];
    const nextState = {
      enabled: input.enabled,
      rolloutPercentage: input.rolloutPercentage,
      targetingRules: input.targetingRules,
      killSwitchActive: input.killSwitchActive,
    };

    const { error: updateError } = await supabase
      .from("flag_environments")
      .update({
        enabled: input.enabled,
        rollout_percentage: input.rolloutPercentage,
        rules: input.targetingRules,
        kill_switch_active: input.killSwitchActive,
        updated_at: new Date().toISOString(),
      })
      .eq("flag_id", input.flagId)
      .eq("environment_id", input.environmentId);
    if (updateError) throw updateError;

    await supabase.from("audit_logs").insert({
      project_id: data.project.id,
      environment_id: input.environmentId,
      action: "flag.updated",
      actor_id: currentUserRow.id,
      actor_email: currentUserRow.email,
      resource_type: "flag",
      resource_id: input.flagId,
      previous_state: previous
        ? {
            enabled: previous.enabled,
            rolloutPercentage: previous.rolloutPercentage,
            targetingRules: previous.targetingRules,
            killSwitchActive: previous.killSwitchActive,
          }
        : null,
      new_state: nextState,
    });

    await reload();
  }

  async function createEnvironment(input: CreateEnvironmentInput) {
    if (!isAuthenticated) {
      throw new Error("Sign in to create persistent environments.");
    }

    const { data: insertedEnvironment, error: environmentError } = await supabase
      .from("environments")
      .insert({
        project_id: data.project.id,
        key: input.key,
        name: input.name,
        color: input.color,
        description: input.description,
        production: input.production,
        sort_order: data.environments.length,
      })
      .select("*")
      .single();
    if (environmentError) throw environmentError;

    if (data.flags.length > 0) {
      const { error: flagEnvironmentError } = await supabase
        .from("flag_environments")
        .insert(
          data.flags.map((flag) => ({
            flag_id: flag.id,
            environment_id: insertedEnvironment.id,
            enabled: false,
            rules: [],
            off_variation: flag.defaultVariation,
            rollout_percentage: 0,
            kill_switch_active: false,
          }))
        );
      if (flagEnvironmentError) throw flagEnvironmentError;
    }

    await supabase.from("audit_logs").insert({
      project_id: data.project.id,
      environment_id: insertedEnvironment.id,
      action: "environment.created",
      actor_id: currentUserRow.id,
      actor_email: currentUserRow.email,
      resource_type: "environment",
      resource_id: insertedEnvironment.id,
      previous_state: null,
      new_state: { key: input.key, name: input.name },
    });

    await reload();
    return insertedEnvironment.id as string;
  }

  return {
    data,
    isLoading: authLoading || isLoading,
    error,
    isAuthenticated,
    isDemo: !isAuthenticated,
    reload,
    createFlag,
    updateFlagEnvironment,
    createEnvironment,
  };
}
