"use client";

import Link from "next/link";
import {
  Flag,
  FlaskConical,
  Globe,
  Activity,
  ArrowRight,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import { AuditTimeline } from "@/components/AuditTimeline";
import { useDemoData } from "@/lib/use-demo-data";

export default function DashboardPage() {
  const { data } = useDemoData();
  const { flags, experiments, environments, auditEntries } = data;
  const productionEnvironment =
    environments.find((environment) => environment.production) ??
    environments[environments.length - 1];
  const stats = [
    {
      label: "Feature Flags",
      value: flags.length,
      icon: Flag,
      color: "text-blue-400",
      bg: "bg-blue-500/10",
      href: "/flags",
    },
    {
      label: "Active Experiments",
      value: experiments.filter((experiment) => experiment.status === "running").length,
      icon: FlaskConical,
      color: "text-purple-400",
      bg: "bg-purple-500/10",
      href: "/experiments",
    },
    {
      label: "Environments",
      value: environments.length,
      icon: Globe,
      color: "text-emerald-400",
      bg: "bg-emerald-500/10",
      href: "/environments",
    },
    {
      label: "Events (24h)",
      value: auditEntries.length,
      icon: Activity,
      color: "text-yellow-400",
      bg: "bg-yellow-500/10",
      href: "/audit",
    },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-zinc-100">Dashboard</h1>
        <p className="mt-1 text-sm text-zinc-400">
          Overview of your feature flag platform
        </p>
      </div>

      {/* Metric cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Link key={stat.label} href={stat.href}>
              <div className="group rounded-xl border border-zinc-800 bg-zinc-900/50 p-5 transition-all hover:border-zinc-700 hover:bg-zinc-900">
                <div className="flex items-center justify-between">
                  <div
                    className={`flex h-10 w-10 items-center justify-center rounded-lg ${stat.bg}`}
                  >
                    <Icon className={`h-5 w-5 ${stat.color}`} />
                  </div>
                  <ArrowRight className="h-4 w-4 text-zinc-700 group-hover:text-zinc-400 transition-colors" />
                </div>
                <div className="mt-4">
                  <p className="text-3xl font-bold text-zinc-100">{stat.value}</p>
                  <p className="mt-1 text-sm text-zinc-500">{stat.label}</p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Recent flags */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-zinc-100">Recent Flags</h2>
            <Link
              href="/flags"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </div>
          <div className="space-y-3">
            {flags.slice(0, 4).map((flag) => {
              const prodEnv = productionEnvironment
                ? flag.environments[productionEnvironment.id]
                : undefined;
              const isEnabled = prodEnv?.enabled ?? false;
              return (
                <Link key={flag.id} href={`/flags/${flag.id}`}>
                  <div className="flex items-center justify-between rounded-lg p-3 hover:bg-zinc-800/50 transition-colors">
                    <div className="flex items-center gap-3">
                      <div
                        className={`h-2 w-2 rounded-full ${
                          isEnabled ? "bg-emerald-500" : "bg-zinc-600"
                        }`}
                      />
                      <div>
                        <p className="text-sm font-medium text-zinc-200">
                          {flag.name}
                        </p>
                        <p className="text-xs text-zinc-500 font-mono">
                          {flag.key}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={flag.type === "boolean" ? "info" : "purple"}>
                        {flag.type}
                      </Badge>
                      {prodEnv && (
                        <span className="text-xs text-zinc-500">
                          {prodEnv.rolloutPercentage}%
                        </span>
                      )}
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        </div>

        {/* Active experiments */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-zinc-100">
              Experiments
            </h2>
            <Link
              href="/experiments"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </div>
          <div className="space-y-3">
            {experiments.slice(0, 4).map((exp) => {
              const statusVariant =
                exp.status === "running"
                  ? "success"
                  : exp.status === "complete"
                  ? "info"
                  : exp.status === "paused"
                  ? "warning"
                  : "default";
              return (
                <Link key={exp.id} href={`/experiments/${exp.id}`}>
                  <div className="flex items-center justify-between rounded-lg p-3 hover:bg-zinc-800/50 transition-colors">
                    <div>
                      <p className="text-sm font-medium text-zinc-200">
                        {exp.name}
                      </p>
                      <p className="text-xs text-zinc-500">
                        {exp.variations.length} variations &middot;{" "}
                        {exp.trafficPercentage}% traffic
                      </p>
                    </div>
                    <Badge variant={statusVariant} dot>
                      {exp.status}
                    </Badge>
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      </div>

      {/* Audit timeline */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-zinc-100">Recent Activity</h2>
          <Link
            href="/audit"
            className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            View full log
          </Link>
        </div>
        <AuditTimeline entries={auditEntries.slice(0, 5)} />
      </div>
    </div>
  );
}
