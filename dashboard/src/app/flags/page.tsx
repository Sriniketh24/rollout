"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { Search, Plus, Filter, X } from "lucide-react";
import { FlagCard } from "@/components/FlagCard";
import { Badge } from "@/components/Badge";
import { mockFlags } from "@/lib/mock-data";
import type { FlagType } from "@/lib/types";

const FLAG_TYPES: FlagType[] = ["boolean", "string", "number", "json"];

export default function FlagsPage() {
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<FlagType | "all">("all");
  const [tagFilter, setTagFilter] = useState<string | null>(null);
  const [showArchived, setShowArchived] = useState(false);

  const allTags = useMemo(() => {
    const tags = new Set<string>();
    mockFlags.forEach((f) => f.tags.forEach((t) => tags.add(t)));
    return Array.from(tags).sort();
  }, []);

  const filtered = useMemo(() => {
    return mockFlags.filter((flag) => {
      if (!showArchived && flag.archived) return false;
      if (
        search &&
        !flag.name.toLowerCase().includes(search.toLowerCase()) &&
        !flag.key.toLowerCase().includes(search.toLowerCase()) &&
        !flag.description.toLowerCase().includes(search.toLowerCase())
      )
        return false;
      if (typeFilter !== "all" && flag.type !== typeFilter) return false;
      if (tagFilter && !flag.tags.includes(tagFilter)) return false;
      return true;
    });
  }, [search, typeFilter, tagFilter, showArchived]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Feature Flags</h1>
          <p className="mt-1 text-sm text-zinc-400">
            {filtered.length} flag{filtered.length !== 1 ? "s" : ""}
          </p>
        </div>
        <Link
          href="/flags/new"
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
        >
          <Plus className="h-4 w-4" />
          New Flag
        </Link>
      </div>

      {/* Search and filters */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
          <input
            type="text"
            placeholder="Search flags by name, key, or description..."
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

        <div className="flex items-center gap-2">
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value as FlagType | "all")}
            className="rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2.5 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
          >
            <option value="all">All types</option>
            {FLAG_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>

          <select
            value={tagFilter ?? ""}
            onChange={(e) => setTagFilter(e.target.value || null)}
            className="rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2.5 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
          >
            <option value="">All tags</option>
            {allTags.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>

          <label className="flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2.5 text-sm text-zinc-300">
            <input
              type="checkbox"
              checked={showArchived}
              onChange={(e) => setShowArchived(e.target.checked)}
              className="h-4 w-4 rounded border-zinc-700 bg-zinc-950 accent-blue-600"
            />
            Archived
          </label>
        </div>
      </div>

      {/* Active filters display */}
      {(typeFilter !== "all" || tagFilter) && (
        <div className="flex items-center gap-2">
          <Filter className="h-3.5 w-3.5 text-zinc-500" />
          <span className="text-xs text-zinc-500">Filters:</span>
          {typeFilter !== "all" && (
            <Badge variant="info">
              type: {typeFilter}
              <button
                onClick={() => setTypeFilter("all")}
                className="ml-1 hover:text-white"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {tagFilter && (
            <Badge variant="purple">
              tag: {tagFilter}
              <button
                onClick={() => setTagFilter(null)}
                className="ml-1 hover:text-white"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
        </div>
      )}

      {/* Flag list */}
      <div className="grid grid-cols-1 gap-4">
        {filtered.map((flag) => (
          <FlagCard key={flag.id} flag={flag} />
        ))}
      </div>

      {filtered.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-zinc-800">
            <Search className="h-6 w-6 text-zinc-500" />
          </div>
          <p className="mt-4 text-sm font-medium text-zinc-300">No flags found</p>
          <p className="mt-1 text-sm text-zinc-500">
            Try adjusting your search or filter criteria
          </p>
        </div>
      )}
    </div>
  );
}
