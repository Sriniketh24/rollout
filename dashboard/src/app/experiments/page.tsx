"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import {
  FlaskConical,
  Play,
  Pause,
  Square,
  Clock,
  TrendingUp,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import { mockExperiments } from "@/lib/mock-data";
import type { ExperimentStatus } from "@/lib/types";

const STATUS_FILTERS: { value: ExperimentStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "running", label: "Running" },
  { value: "complete", label: "Complete" },
  { value: "paused", label: "Paused" },
  { value: "draft", label: "Draft" },
  { value: "stopped", label: "Stopped" },
];

function getStatusConfig(status: ExperimentStatus) {
  const map: Record<
    ExperimentStatus,
    {
      variant: "success" | "info" | "warning" | "danger" | "default";
      icon: React.ElementType;
    }
  > = {
    running: { variant: "success", icon: Play },
    complete: { variant: "info", icon: TrendingUp },
    paused: { variant: "warning", icon: Pause },
    stopped: { variant: "danger", icon: Square },
    draft: { variant: "default", icon: Clock },
  };
  return map[status];
}

export default function ExperimentsPage() {
  const [statusFilter, setStatusFilter] = useState<ExperimentStatus | "all">(
    "all"
  );

  const filtered = useMemo(() => {
    if (statusFilter === "all") return mockExperiments;
    return mockExperiments.filter((e) => e.status === statusFilter);
  }, [statusFilter]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Experiments</h1>
          <p className="mt-1 text-sm text-zinc-400">
            {filtered.length} experiment{filtered.length !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      {/* Status filter tabs */}
      <div className="flex gap-1 rounded-lg bg-zinc-900/50 border border-zinc-800 p-1">
        {STATUS_FILTERS.map((sf) => (
          <button
            key={sf.value}
            onClick={() => setStatusFilter(sf.value)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              statusFilter === sf.value
                ? "bg-zinc-800 text-zinc-100"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            {sf.label}
          </button>
        ))}
      </div>

      {/* Experiment cards */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {filtered.map((exp) => {
          const statusConfig = getStatusConfig(exp.status);

          return (
            <Link key={exp.id} href={`/experiments/${exp.id}`}>
              <div className="group rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 transition-all hover:border-zinc-700 hover:bg-zinc-900">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400">
                      <FlaskConical className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-zinc-100">
                        {exp.name}
                      </h3>
                      <p className="mt-0.5 font-mono text-xs text-zinc-500">
                        {exp.key}
                      </p>
                    </div>
                  </div>
                  <Badge variant={statusConfig.variant} dot>
                    {exp.status}
                  </Badge>
                </div>

                <p className="mt-3 text-sm text-zinc-400 line-clamp-2">
                  {exp.description}
                </p>

                <div className="mt-4 grid grid-cols-3 gap-4 border-t border-zinc-800 pt-4">
                  <div>
                    <p className="text-xs text-zinc-500">Variations</p>
                    <p className="mt-0.5 text-sm font-medium text-zinc-200">
                      {exp.variations.length}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-zinc-500">Traffic</p>
                    <p className="mt-0.5 text-sm font-medium text-zinc-200">
                      {exp.trafficPercentage}%
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-zinc-500">Metric</p>
                    <p className="mt-0.5 text-sm font-medium text-zinc-200 truncate">
                      {exp.targetMetric.replace(/_/g, " ")}
                    </p>
                  </div>
                </div>

                {exp.startDate && (
                  <div className="mt-3 flex items-center gap-1.5 text-xs text-zinc-500">
                    <Clock className="h-3 w-3" />
                    Started{" "}
                    {new Date(exp.startDate).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                    {exp.endDate &&
                      ` -- Ended ${new Date(exp.endDate).toLocaleDateString(
                        "en-US",
                        { month: "short", day: "numeric", year: "numeric" }
                      )}`}
                  </div>
                )}
              </div>
            </Link>
          );
        })}
      </div>

      {filtered.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-zinc-800">
            <FlaskConical className="h-6 w-6 text-zinc-500" />
          </div>
          <p className="mt-4 text-sm font-medium text-zinc-300">
            No experiments found
          </p>
          <p className="mt-1 text-sm text-zinc-500">
            No experiments match the selected filter
          </p>
        </div>
      )}
    </div>
  );
}
