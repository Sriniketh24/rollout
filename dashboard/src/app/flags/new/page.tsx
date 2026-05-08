"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, Plus, Trash2, Flag } from "lucide-react";
import type { FlagType } from "@/lib/types";

const FLAG_TYPES: { value: FlagType; label: string; description: string }[] = [
  { value: "boolean", label: "Boolean", description: "Simple on/off toggle" },
  { value: "string", label: "String", description: "Text-based variations" },
  { value: "number", label: "Number", description: "Numeric variations" },
  { value: "json", label: "JSON", description: "Complex configuration objects" },
];

interface Variation {
  key: string;
  value: string;
  name: string;
}

export default function NewFlagPage() {
  const router = useRouter();
  const [key, setKey] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState<FlagType>("boolean");
  const [tags, setTags] = useState("");
  const [variations, setVariations] = useState<Variation[]>([
    { key: "on", value: "true", name: "Enabled" },
    { key: "off", value: "false", name: "Disabled" },
  ]);
  const [defaultVariation, setDefaultVariation] = useState("off");

  function handleNameChange(val: string) {
    setName(val);
    if (!key || key === nameToKey(name)) {
      setKey(nameToKey(val));
    }
  }

  function nameToKey(n: string) {
    return n
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "");
  }

  function addVariation() {
    setVariations([...variations, { key: "", value: "", name: "" }]);
  }

  function removeVariation(index: number) {
    setVariations(variations.filter((_, i) => i !== index));
  }

  function updateVariation(index: number, field: keyof Variation, val: string) {
    setVariations(
      variations.map((v, i) => (i === index ? { ...v, [field]: val } : v))
    );
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    router.push("/flags");
  }

  return (
    <div className="space-y-8 max-w-3xl">
      <Link
        href="/flags"
        className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to flags
      </Link>

      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10">
          <Flag className="h-6 w-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100">Create Flag</h1>
          <p className="text-sm text-zinc-400">
            Define a new feature flag for your application
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic info */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-4">
          <h2 className="text-lg font-semibold text-zinc-100">Basic Information</h2>

          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="e.g. Dark Mode"
              required
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Key
            </label>
            <input
              type="text"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="e.g. dark-mode"
              required
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 font-mono placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
            />
            <p className="mt-1 text-xs text-zinc-500">
              Used in code to reference this flag. Cannot be changed later.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does this flag control?"
              rows={3}
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30 resize-none"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-zinc-300 mb-1.5">
              Tags
            </label>
            <input
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="frontend, ui, experiment (comma separated)"
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
            />
          </div>
        </div>

        {/* Type selection */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-4">
          <h2 className="text-lg font-semibold text-zinc-100">Flag Type</h2>
          <div className="grid grid-cols-2 gap-3">
            {FLAG_TYPES.map((ft) => (
              <button
                key={ft.value}
                type="button"
                onClick={() => setType(ft.value)}
                className={`rounded-xl border p-4 text-left transition-all ${
                  type === ft.value
                    ? "border-blue-500/50 bg-blue-500/5 ring-1 ring-blue-500/20"
                    : "border-zinc-800 bg-zinc-950 hover:border-zinc-700"
                }`}
              >
                <p className="text-sm font-semibold text-zinc-200">{ft.label}</p>
                <p className="mt-0.5 text-xs text-zinc-500">{ft.description}</p>
              </button>
            ))}
          </div>
        </div>

        {/* Variations */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-4">
          <h2 className="text-lg font-semibold text-zinc-100">Variations</h2>

          <div className="space-y-3">
            {variations.map((v, idx) => (
              <div
                key={idx}
                className="flex items-start gap-3 rounded-lg border border-zinc-800 bg-zinc-950 p-3"
              >
                <div className="flex-1 grid grid-cols-3 gap-3">
                  <div>
                    <label className="block text-xs text-zinc-500 mb-1">Key</label>
                    <input
                      type="text"
                      value={v.key}
                      onChange={(e) => updateVariation(idx, "key", e.target.value)}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-sm text-zinc-100 font-mono focus:border-blue-500/50 focus:outline-none"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-zinc-500 mb-1">Name</label>
                    <input
                      type="text"
                      value={v.name}
                      onChange={(e) => updateVariation(idx, "name", e.target.value)}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-zinc-500 mb-1">Value</label>
                    <input
                      type="text"
                      value={v.value}
                      onChange={(e) => updateVariation(idx, "value", e.target.value)}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-sm text-zinc-100 font-mono focus:border-blue-500/50 focus:outline-none"
                    />
                  </div>
                </div>
                <div className="flex items-center gap-2 pt-5">
                  <input
                    type="radio"
                    name="defaultVariation"
                    checked={defaultVariation === v.key}
                    onChange={() => setDefaultVariation(v.key)}
                    className="accent-blue-500"
                    title="Set as default"
                  />
                  <button
                    type="button"
                    onClick={() => removeVariation(idx)}
                    disabled={variations.length <= 2}
                    className="text-zinc-600 hover:text-red-400 disabled:opacity-30 transition-colors"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>

          <button
            type="button"
            onClick={addVariation}
            className="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Add variation
          </button>
        </div>

        {/* Submit */}
        <div className="flex items-center justify-end gap-3">
          <Link
            href="/flags"
            className="rounded-lg px-4 py-2.5 text-sm font-medium text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            Cancel
          </Link>
          <button
            type="submit"
            className="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
          >
            Create Flag
          </button>
        </div>
      </form>
    </div>
  );
}
