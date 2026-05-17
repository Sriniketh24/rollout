"use client";

import { useState } from "react";
import {
  Globe,
  Plus,
  Shield,
  Code,
  Server,
  Pencil,
  Trash2,
  X,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import type { Environment } from "@/lib/types";
import { useDemoData } from "@/lib/use-demo-data";

export default function EnvironmentsPage() {
  const { data, createEnvironment, isAuthenticated } = useDemoData();
  const [draftEnvironments, setDraftEnvironments] =
    useState<Environment[] | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newKey, setNewKey] = useState("");
  const [newColor, setNewColor] = useState("#3b82f6");
  const [newDescription, setNewDescription] = useState("");
  const [newProduction, setNewProduction] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const environments = draftEnvironments ?? data.environments;

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (isAuthenticated) {
      setIsSubmitting(true);
      try {
        await createEnvironment({
          key: newKey,
          name: newName,
          color: newColor,
          description: newDescription,
          production: newProduction,
        });
        setShowCreate(false);
        setNewName("");
        setNewKey("");
        setNewColor("#3b82f6");
        setNewDescription("");
        setNewProduction(false);
      } catch (submitError) {
        setError(
          submitError instanceof Error
            ? submitError.message
            : "Unable to create environment"
        );
      } finally {
        setIsSubmitting(false);
      }
      return;
    }

    const env: Environment = {
      id: `env-${Date.now()}`,
      key: newKey,
      name: newName,
      color: newColor,
      description: newDescription,
      production: newProduction,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      projectId: "proj-1",
      order: environments.length,
    };
    setDraftEnvironments([...(draftEnvironments ?? data.environments), env]);
    setShowCreate(false);
    setNewName("");
    setNewKey("");
    setNewColor("#3b82f6");
    setNewDescription("");
    setNewProduction(false);
  }

  function getFlagCount(envId: string) {
    return data.flags.filter((flag) => flag.environments[envId]?.enabled).length;
  }

  const envIcons: Record<string, React.ElementType> = {
    development: Code,
    staging: Server,
    production: Shield,
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Environments</h1>
          <p className="mt-1 text-sm text-zinc-400">
            Manage deployment environments for your feature flags
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
        >
          <Plus className="h-4 w-4" />
          New Environment
        </button>
      </div>

      {/* Environment cards */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        {environments.map((env) => {
          const Icon = envIcons[env.key] ?? Globe;
          const flagCount = getFlagCount(env.id);

          return (
            <div
              key={env.id}
              className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 transition-all hover:border-zinc-700"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className="flex h-10 w-10 items-center justify-center rounded-lg"
                    style={{ backgroundColor: `${env.color}15` }}
                  >
                    <Icon className="h-5 w-5" style={{ color: env.color }} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-zinc-100">
                        {env.name}
                      </h3>
                      {env.production && (
                        <Badge variant="danger" dot>
                          production
                        </Badge>
                      )}
                    </div>
                    <p className="font-mono text-xs text-zinc-500">{env.key}</p>
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  <button className="rounded-md p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors">
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  {!env.production && (
                    <button className="rounded-md p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-800 transition-colors">
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              </div>

              {env.description && (
                <p className="mt-3 text-sm text-zinc-400">{env.description}</p>
              )}

              <div className="mt-4 grid grid-cols-2 gap-4 border-t border-zinc-800 pt-4">
                <div>
                  <p className="text-xs text-zinc-500">Active Flags</p>
                  <p className="mt-0.5 text-lg font-semibold text-zinc-200">
                    {flagCount}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-zinc-500">Order</p>
                  <p className="mt-0.5 text-lg font-semibold text-zinc-200">
                    {env.order + 1}
                  </p>
                </div>
              </div>

              <div
                className="mt-4 h-1 rounded-full"
                style={{ backgroundColor: env.color }}
              />
            </div>
          );
        })}
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-900 p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-zinc-100">
                New Environment
              </h3>
              <button
                onClick={() => setShowCreate(false)}
                className="text-zinc-500 hover:text-zinc-300"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-300 mb-1.5">
                  Name
                </label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => {
                    setNewName(e.target.value);
                    setNewKey(
                      e.target.value
                        .toLowerCase()
                        .replace(/[^a-z0-9]+/g, "-")
                        .replace(/^-|-$/g, "")
                    );
                  }}
                  required
                  placeholder="e.g. Canary"
                  className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-zinc-300 mb-1.5">
                  Key
                </label>
                <input
                  type="text"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  required
                  className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 font-mono focus:border-blue-500/50 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-zinc-300 mb-1.5">
                  Description
                </label>
                <input
                  type="text"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="Optional description"
                  className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none"
                />
              </div>
              <div className="flex items-center gap-4">
                <div>
                  <label className="block text-sm font-medium text-zinc-300 mb-1.5">
                    Color
                  </label>
                  <input
                    type="color"
                    value={newColor}
                    onChange={(e) => setNewColor(e.target.value)}
                    className="h-10 w-14 rounded-lg border border-zinc-800 bg-zinc-950 cursor-pointer"
                  />
                </div>
                <div className="flex-1">
                  <label className="flex items-center gap-2 mt-4 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={newProduction}
                      onChange={(e) => setNewProduction(e.target.checked)}
                      className="rounded border-zinc-700 accent-red-500"
                    />
                    <span className="text-sm text-zinc-300">
                      Production environment
                    </span>
                  </label>
                </div>
              </div>

              <div className="flex justify-end gap-3 mt-6">
                {error && (
                  <div className="mr-auto max-w-52 text-sm text-red-300">
                    {error}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => setShowCreate(false)}
                  className="rounded-lg px-4 py-2 text-sm font-medium text-zinc-400 hover:text-zinc-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isSubmitting ? "Creating..." : "Create"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
