"use client";

import {
  Flag,
  FlaskConical,
  Globe,
  UserPlus,
  UserMinus,
  Settings,
  ShieldOff,
  ToggleRight,
  Target,
  Gauge,
} from "lucide-react";
import { Badge } from "./Badge";
import type { AuditEntry, AuditAction } from "@/lib/types";

function getActionConfig(action: AuditAction) {
  const map: Record<
    AuditAction,
    { icon: React.ElementType; label: string; color: string; badgeVariant: "info" | "success" | "warning" | "danger" | "purple" | "default" }
  > = {
    "flag.created": { icon: Flag, label: "Flag Created", color: "text-blue-400", badgeVariant: "info" },
    "flag.updated": { icon: Flag, label: "Flag Updated", color: "text-blue-400", badgeVariant: "info" },
    "flag.deleted": { icon: Flag, label: "Flag Deleted", color: "text-red-400", badgeVariant: "danger" },
    "flag.toggled": { icon: ToggleRight, label: "Flag Toggled", color: "text-emerald-400", badgeVariant: "success" },
    "flag.killed": { icon: ShieldOff, label: "Kill Switch", color: "text-red-400", badgeVariant: "danger" },
    "experiment.created": { icon: FlaskConical, label: "Experiment Created", color: "text-purple-400", badgeVariant: "purple" },
    "experiment.started": { icon: FlaskConical, label: "Experiment Started", color: "text-emerald-400", badgeVariant: "success" },
    "experiment.paused": { icon: FlaskConical, label: "Experiment Paused", color: "text-yellow-400", badgeVariant: "warning" },
    "experiment.stopped": { icon: FlaskConical, label: "Experiment Stopped", color: "text-red-400", badgeVariant: "danger" },
    "experiment.completed": { icon: FlaskConical, label: "Experiment Completed", color: "text-emerald-400", badgeVariant: "success" },
    "environment.created": { icon: Globe, label: "Environment Created", color: "text-blue-400", badgeVariant: "info" },
    "environment.updated": { icon: Globe, label: "Environment Updated", color: "text-blue-400", badgeVariant: "info" },
    "environment.deleted": { icon: Globe, label: "Environment Deleted", color: "text-red-400", badgeVariant: "danger" },
    "targeting.updated": { icon: Target, label: "Targeting Updated", color: "text-purple-400", badgeVariant: "purple" },
    "rollout.updated": { icon: Gauge, label: "Rollout Updated", color: "text-yellow-400", badgeVariant: "warning" },
    "project.updated": { icon: Settings, label: "Project Updated", color: "text-blue-400", badgeVariant: "info" },
    "user.invited": { icon: UserPlus, label: "User Invited", color: "text-emerald-400", badgeVariant: "success" },
    "user.removed": { icon: UserMinus, label: "User Removed", color: "text-red-400", badgeVariant: "danger" },
  };
  return map[action] ?? { icon: Settings, label: action, color: "text-zinc-400", badgeVariant: "default" as const };
}

function formatRelativeTime(timestamp: string) {
  const now = new Date();
  const date = new Date(timestamp);
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

interface AuditTimelineProps {
  entries: AuditEntry[];
}

export function AuditTimeline({ entries }: AuditTimelineProps) {
  return (
    <div className="relative">
      <div className="absolute left-5 top-0 bottom-0 w-px bg-zinc-800" />

      <div className="space-y-1">
        {entries.map((entry) => {
          const config = getActionConfig(entry.action);
          const Icon = config.icon;
          const environment = entry.metadata?.environment;

          return (
            <div key={entry.id} className="relative flex gap-4 py-3 pl-1">
              <div
                className={`relative z-10 flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-zinc-800 bg-zinc-900 ${config.color}`}
              >
                <Icon className="h-4 w-4" />
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-medium text-zinc-200">
                      {entry.actor}
                    </span>
                    <Badge variant={config.badgeVariant}>{config.label}</Badge>
                  </div>
                  <span className="shrink-0 text-xs text-zinc-500">
                    {formatRelativeTime(entry.timestamp)}
                  </span>
                </div>

                <p className="mt-0.5 text-sm text-zinc-400">
                  {entry.targetName && (
                    <span className="font-medium text-zinc-300">
                      {entry.targetName}
                    </span>
                  )}
                  {environment != null && (
                    <span className="text-zinc-500">
                      {" "}
                      in {String(environment)}
                    </span>
                  )}
                </p>

                {entry.changes && (
                  <div className="mt-1.5 rounded-lg bg-zinc-950 border border-zinc-800 px-3 py-2">
                    {Object.entries(entry.changes).map(([key, change]) => (
                      <div key={key} className="flex items-center gap-2 text-xs font-mono">
                        <span className="text-zinc-500">{key}:</span>
                        <span className="text-red-400 line-through">
                          {JSON.stringify(change.before)}
                        </span>
                        <span className="text-zinc-600">-&gt;</span>
                        <span className="text-emerald-400">
                          {JSON.stringify(change.after)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default AuditTimeline;
