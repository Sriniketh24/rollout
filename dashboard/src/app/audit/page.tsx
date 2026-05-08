"use client";

import { useState, useMemo } from "react";
import { Search, X, Filter } from "lucide-react";
import { AuditTimeline } from "@/components/AuditTimeline";
import { useDemoData } from "@/lib/use-demo-data";

const ACTION_GROUPS = [
  { label: "All", value: "all" },
  { label: "Flags", value: "flag" },
  { label: "Experiments", value: "experiment" },
  { label: "Environments", value: "environment" },
  { label: "Targeting", value: "targeting" },
  { label: "Rollout", value: "rollout" },
  { label: "Users", value: "user" },
  { label: "Project", value: "project" },
];

export default function AuditPage() {
  const { data } = useDemoData();
  const [search, setSearch] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const entries = data.auditEntries;

  const filtered = useMemo(() => {
    return entries.filter((entry) => {
      if (
        search &&
        !entry.actor.toLowerCase().includes(search.toLowerCase()) &&
        !(entry.targetName ?? "")
          .toLowerCase()
          .includes(search.toLowerCase()) &&
        !entry.action.toLowerCase().includes(search.toLowerCase())
      )
        return false;

      if (actionFilter !== "all" && !entry.action.startsWith(actionFilter))
        return false;

      return true;
    });
  }, [entries, search, actionFilter]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-zinc-100">Audit Log</h1>
        <p className="mt-1 text-sm text-zinc-400">
          Track all changes across your feature flag platform
        </p>
      </div>

      {/* Search */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
          <input
            type="text"
            placeholder="Search by actor, target, or action..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-zinc-800 bg-zinc-900 pl-10 pr-4 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-500 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
          />
          {search && (
            <button
              onClick={() => setSearch("")}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-1 rounded-lg bg-zinc-900/50 border border-zinc-800 p-1 flex-wrap">
        {ACTION_GROUPS.map((group) => (
          <button
            key={group.value}
            onClick={() => setActionFilter(group.value)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              actionFilter === group.value
                ? "bg-zinc-800 text-zinc-100"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            {group.label}
          </button>
        ))}
      </div>

      {/* Results count */}
      <div className="flex items-center gap-2 text-sm text-zinc-500">
        <Filter className="h-3.5 w-3.5" />
        <span>
          {filtered.length} event{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Timeline */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
        {filtered.length > 0 ? (
          <AuditTimeline entries={filtered} />
        ) : (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Search className="h-8 w-8 text-zinc-600 mb-3" />
            <p className="text-sm text-zinc-400">No audit entries found</p>
            <p className="text-xs text-zinc-500 mt-1">
              Try adjusting your search or filter
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
