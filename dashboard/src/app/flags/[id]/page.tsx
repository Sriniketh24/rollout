"use client";

import { useState, use } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Flag as FlagIcon,
  Clock,
  User,
  Tag,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import { RolloutSlider } from "@/components/RolloutSlider";
import { KillSwitchButton } from "@/components/KillSwitchButton";
import { TargetingRuleEditor } from "@/components/TargetingRuleEditor";
import type { TargetingRule } from "@/lib/types";
import { useDemoData } from "@/lib/use-demo-data";

interface EnvironmentDraft {
  enabled: boolean;
  rollout: number;
  killActive: boolean;
  rules: TargetingRule[];
}

export default function FlagDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data } = useDemoData();
  const flag = data.flags.find((entry) => entry.id === id) ?? data.flags[0];

  const [selectedEnv, setSelectedEnv] = useState("env-3");
  const [draftsByEnvironment, setDraftsByEnvironment] = useState<
    Record<string, EnvironmentDraft>
  >({});

  const resolvedSelectedEnv =
    flag?.environments[selectedEnv] != null
      ? selectedEnv
      : data.environments[0]?.id ?? "";
  const envConfig = flag?.environments[resolvedSelectedEnv];
  const envMeta = data.environments.find(
    (environment) => environment.id === resolvedSelectedEnv
  );

  function baseDraft(): EnvironmentDraft {
    return {
      enabled: envConfig?.enabled ?? false,
      rollout: envConfig?.rolloutPercentage ?? 0,
      killActive: envConfig?.killSwitchActive ?? false,
      rules: envConfig?.targetingRules ?? [],
    };
  }

  const activeDraft = draftsByEnvironment[resolvedSelectedEnv] ?? baseDraft();

  function updateDraft(partial: Partial<EnvironmentDraft>) {
    setDraftsByEnvironment((current) => ({
      ...current,
      [resolvedSelectedEnv]: {
        ...(current[resolvedSelectedEnv] ?? baseDraft()),
        ...partial,
      },
    }));
  }

  function handleEnvSwitch(envId: string) {
    setSelectedEnv(envId);
  }

  return (
    <div className="space-y-8">
      {/* Back nav */}
      <Link
        href="/flags"
        className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to flags
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10">
            <FlagIcon className="h-6 w-6 text-blue-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-zinc-100">{flag.name}</h1>
            <p className="mt-0.5 font-mono text-sm text-zinc-500">{flag.key}</p>
            <p className="mt-2 text-sm text-zinc-400">{flag.description}</p>
            <div className="mt-3 flex items-center gap-3">
              <Badge
                variant={
                  flag.type === "boolean"
                    ? "info"
                    : flag.type === "string"
                    ? "purple"
                    : "warning"
                }
              >
                {flag.type}
              </Badge>
              {flag.tags.map((tag) => (
                <Badge key={tag} variant="outline">
                  <Tag className="h-3 w-3 mr-1" />
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        </div>

        <div className="text-right text-xs text-zinc-500 space-y-1">
          <div className="flex items-center justify-end gap-1.5">
            <Clock className="h-3.5 w-3.5" />
            Created {new Date(flag.createdAt).toLocaleDateString()}
          </div>
          <div className="flex items-center justify-end gap-1.5">
            <User className="h-3.5 w-3.5" />
            {flag.createdBy}
          </div>
        </div>
      </div>

      {/* Environment tabs */}
      <div className="flex gap-2 border-b border-zinc-800 pb-0">
        {data.environments.map((env) => (
          <button
            key={env.id}
            onClick={() => handleEnvSwitch(env.id)}
            className={`relative px-4 py-2.5 text-sm font-medium transition-colors ${
              selectedEnv === env.id
                ? "text-zinc-100"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            <div className="flex items-center gap-2">
              <div
                className="h-2 w-2 rounded-full"
                style={{ backgroundColor: env.color }}
              />
              {env.name}
            </div>
            {selectedEnv === env.id && (
              <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 rounded-full" />
            )}
          </button>
        ))}
      </div>

      {envConfig && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Main content */}
          <div className="lg:col-span-2 space-y-6">
            {/* Toggle and kill switch */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-semibold text-zinc-100">
                    Flag Status
                  </h2>
                  <p className="mt-0.5 text-sm text-zinc-400">
                    Toggle this flag in{" "}
                    <span className="font-medium text-zinc-300">
                      {envMeta?.name ?? selectedEnv}
                    </span>
                  </p>
                </div>
                <div className="flex items-center gap-4">
                  <KillSwitchButton
                    flagName={flag.name}
                    environmentName={envMeta?.name ?? resolvedSelectedEnv}
                    isActive={activeDraft.killActive}
                    onActivate={() => updateDraft({ killActive: true })}
                    onDeactivate={() => updateDraft({ killActive: false })}
                  />
                  <button
                    onClick={() => updateDraft({ enabled: !activeDraft.enabled })}
                    className="transition-colors"
                  >
                    {activeDraft.enabled ? (
                      <ToggleRight className="h-10 w-10 text-emerald-500" />
                    ) : (
                      <ToggleLeft className="h-10 w-10 text-zinc-500" />
                    )}
                  </button>
                </div>
              </div>

              {activeDraft.killActive && (
                <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-300">
                  Kill switch is active. All users are receiving the off
                  variation.
                </div>
              )}
            </div>

            {/* Rollout slider */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <h2 className="text-lg font-semibold text-zinc-100 mb-4">
                Rollout Percentage
              </h2>
              <RolloutSlider
                value={activeDraft.rollout}
                onChange={(rollout) => updateDraft({ rollout })}
                disabled={!activeDraft.enabled || activeDraft.killActive}
              />
            </div>

            {/* Targeting rules */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <h2 className="text-lg font-semibold text-zinc-100 mb-1">
                Targeting Rules
              </h2>
              <p className="text-sm text-zinc-400 mb-4">
                Define rules to serve specific variations to targeted users
              </p>
              <TargetingRuleEditor
                rules={activeDraft.rules}
                variations={flag.variations}
                onChange={(rules) => updateDraft({ rules })}
              />
            </div>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Variations */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <h3 className="text-sm font-semibold text-zinc-100 mb-3">
                Variations
              </h3>
              <div className="space-y-2">
                {flag.variations.map((v) => (
                  <div
                    key={v.key}
                    className={`flex items-center justify-between rounded-lg border px-3 py-2.5 ${
                      v.key === flag.defaultVariation
                        ? "border-blue-500/30 bg-blue-500/5"
                        : "border-zinc-800 bg-zinc-950"
                    }`}
                  >
                    <div>
                      <p className="text-sm font-medium text-zinc-200">
                        {v.name ?? v.key}
                      </p>
                      <p className="text-xs text-zinc-500 font-mono">
                        {JSON.stringify(v.value)}
                      </p>
                    </div>
                    {v.key === flag.defaultVariation && (
                      <Badge variant="info">default</Badge>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Environment status */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <h3 className="text-sm font-semibold text-zinc-100 mb-3">
                All Environments
              </h3>
              <div className="space-y-2">
                {data.environments.map((env) => {
                  const cfg = flag.environments[env.id];
                  return (
                    <div
                      key={env.id}
                      className="flex items-center justify-between rounded-lg px-3 py-2"
                    >
                      <div className="flex items-center gap-2">
                        <div
                          className="h-2 w-2 rounded-full"
                          style={{ backgroundColor: env.color }}
                        />
                        <span className="text-sm text-zinc-300">{env.name}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        {cfg?.enabled ? (
                          <Badge variant="success" dot>
                            {cfg.rolloutPercentage}%
                          </Badge>
                        ) : (
                          <Badge variant="default">off</Badge>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Metadata */}
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
              <h3 className="text-sm font-semibold text-zinc-100 mb-3">
                Details
              </h3>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-zinc-500">Last modified</dt>
                  <dd className="text-zinc-300">
                    {new Date(envConfig.lastModified).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-zinc-500">Rules</dt>
                  <dd className="text-zinc-300">
                    {activeDraft.rules.length}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-zinc-500">Kill switch</dt>
                  <dd>
                    {activeDraft.killActive ? (
                      <Badge variant="danger">active</Badge>
                    ) : (
                      <span className="text-zinc-300">inactive</span>
                    )}
                  </dd>
                </div>
              </dl>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
