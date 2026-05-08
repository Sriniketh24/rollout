"use client";

import { useState } from "react";
import Link from "next/link";
import { Flag as FlagIcon, ToggleLeft, ToggleRight, ChevronRight } from "lucide-react";
import { Badge } from "./Badge";
import type { Flag } from "@/lib/types";

interface FlagCardProps {
  flag: Flag;
  onToggle?: (flagId: string, envId: string, enabled: boolean) => void;
}

export function FlagCard({ flag, onToggle }: FlagCardProps) {
  const envEntries = Object.entries(flag.environments);
  const prodEnv = envEntries.find(([, cfg]) => cfg.environmentId === "env-3");
  const isProdEnabled = prodEnv ? prodEnv[1].enabled : false;

  const [toggling, setToggling] = useState(false);

  function handleToggle(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (!prodEnv || toggling) return;
    setToggling(true);
    onToggle?.(flag.id, prodEnv[1].environmentId, !isProdEnabled);
    setTimeout(() => setToggling(false), 300);
  }

  const typeBadgeVariant =
    flag.type === "boolean"
      ? "info"
      : flag.type === "string"
      ? "purple"
      : flag.type === "number"
      ? "warning"
      : "default";

  return (
    <Link href={`/flags/${flag.id}`}>
      <div className="group rounded-xl border border-zinc-800 bg-zinc-900/50 p-5 transition-all hover:border-zinc-700 hover:bg-zinc-900">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-9 w-9 items-center justify-center rounded-lg bg-zinc-800 text-zinc-400 group-hover:bg-blue-500/10 group-hover:text-blue-400 transition-colors">
              <FlagIcon className="h-4 w-4" />
            </div>
            <div>
              <h3 className="font-semibold text-zinc-100">{flag.name}</h3>
              <p className="mt-0.5 font-mono text-xs text-zinc-500">{flag.key}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleToggle}
              className="text-zinc-400 hover:text-zinc-200 transition-colors"
              title={isProdEnabled ? "Disable in production" : "Enable in production"}
            >
              {isProdEnabled ? (
                <ToggleRight className="h-7 w-7 text-emerald-500" />
              ) : (
                <ToggleLeft className="h-7 w-7" />
              )}
            </button>
            <ChevronRight className="h-4 w-4 text-zinc-600 group-hover:text-zinc-400 transition-colors" />
          </div>
        </div>

        <p className="mt-3 text-sm text-zinc-400 line-clamp-2">{flag.description}</p>

        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Badge variant={typeBadgeVariant}>{flag.type}</Badge>
            {flag.tags.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="outline">
                {tag}
              </Badge>
            ))}
          </div>
          <div className="flex items-center gap-2">
            {envEntries.map(([, cfg]) => (
              <div
                key={cfg.environmentId}
                className={`h-2 w-2 rounded-full ${
                  cfg.enabled ? "bg-emerald-500" : "bg-zinc-600"
                }`}
                title={`${cfg.environmentId}: ${cfg.enabled ? "enabled" : "disabled"}`}
              />
            ))}
          </div>
        </div>
      </div>
    </Link>
  );
}

export default FlagCard;
