"use client";

import { useState } from "react";
import { Plus, Trash2, GripVertical } from "lucide-react";
import type { TargetingRule, Operator, Clause } from "@/lib/types";

const OPERATORS: { value: Operator; label: string }[] = [
  { value: "eq", label: "equals" },
  { value: "neq", label: "not equals" },
  { value: "in", label: "is in" },
  { value: "not_in", label: "is not in" },
  { value: "contains", label: "contains" },
  { value: "not_contains", label: "not contains" },
  { value: "starts_with", label: "starts with" },
  { value: "ends_with", label: "ends with" },
  { value: "gt", label: ">" },
  { value: "gte", label: ">=" },
  { value: "lt", label: "<" },
  { value: "lte", label: "<=" },
  { value: "exists", label: "exists" },
  { value: "not_exists", label: "not exists" },
  { value: "matches", label: "matches regex" },
];

interface TargetingRuleEditorProps {
  rules: TargetingRule[];
  variations: { key: string; name?: string }[];
  onChange: (rules: TargetingRule[]) => void;
}

export function TargetingRuleEditor({
  rules,
  variations,
  onChange,
}: TargetingRuleEditorProps) {
  const [localRules, setLocalRules] = useState<TargetingRule[]>(rules);

  function updateRules(updated: TargetingRule[]) {
    setLocalRules(updated);
    onChange(updated);
  }

  function addRule() {
    const newRule: TargetingRule = {
      id: `rule-${Date.now()}`,
      description: "",
      clauses: [{ attribute: "", operator: "eq", values: [""] }],
      variation: variations[0]?.key ?? "",
      rolloutPercentage: 100,
      priority: localRules.length,
    };
    updateRules([...localRules, newRule]);
  }

  function removeRule(index: number) {
    updateRules(localRules.filter((_, i) => i !== index));
  }

  function updateRule(index: number, partial: Partial<TargetingRule>) {
    const updated = localRules.map((r, i) =>
      i === index ? { ...r, ...partial } : r
    );
    updateRules(updated);
  }

  function addClause(ruleIndex: number) {
    const rule = localRules[ruleIndex];
    const newClause: Clause = { attribute: "", operator: "eq", values: [""] };
    updateRule(ruleIndex, { clauses: [...rule.clauses, newClause] });
  }

  function removeClause(ruleIndex: number, clauseIndex: number) {
    const rule = localRules[ruleIndex];
    updateRule(ruleIndex, {
      clauses: rule.clauses.filter((_, i) => i !== clauseIndex),
    });
  }

  function updateClause(
    ruleIndex: number,
    clauseIndex: number,
    partial: Partial<Clause>
  ) {
    const rule = localRules[ruleIndex];
    const clauses = rule.clauses.map((c, i) =>
      i === clauseIndex ? { ...c, ...partial } : c
    );
    updateRule(ruleIndex, { clauses });
  }

  return (
    <div className="space-y-4">
      {localRules.map((rule, ruleIdx) => (
        <div
          key={rule.id}
          className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4"
        >
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <GripVertical className="h-4 w-4 text-zinc-600 cursor-grab" />
              <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider">
                Rule {ruleIdx + 1}
              </span>
            </div>
            <button
              onClick={() => removeRule(ruleIdx)}
              className="text-zinc-600 hover:text-red-400 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>

          <input
            type="text"
            placeholder="Rule description (optional)"
            value={rule.description ?? ""}
            onChange={(e) =>
              updateRule(ruleIdx, { description: e.target.value })
            }
            className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none mb-3"
          />

          <div className="space-y-2">
            {rule.clauses.map((clause, clauseIdx) => (
              <div
                key={clauseIdx}
                className="flex items-center gap-2"
              >
                {clauseIdx > 0 && (
                  <span className="text-xs font-medium text-blue-400 w-8 text-center">
                    AND
                  </span>
                )}
                {clauseIdx === 0 && <span className="w-8" />}
                <input
                  type="text"
                  placeholder="attribute"
                  value={clause.attribute}
                  onChange={(e) =>
                    updateClause(ruleIdx, clauseIdx, {
                      attribute: e.target.value,
                    })
                  }
                  className="w-36 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none font-mono"
                />
                <select
                  value={clause.operator}
                  onChange={(e) =>
                    updateClause(ruleIdx, clauseIdx, {
                      operator: e.target.value as Operator,
                    })
                  }
                  className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
                >
                  {OPERATORS.map((op) => (
                    <option key={op.value} value={op.value}>
                      {op.label}
                    </option>
                  ))}
                </select>
                <input
                  type="text"
                  placeholder="value1, value2"
                  value={clause.values.join(", ")}
                  onChange={(e) =>
                    updateClause(ruleIdx, clauseIdx, {
                      values: e.target.value.split(",").map((v) => v.trim()),
                    })
                  }
                  className="flex-1 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none font-mono"
                />
                <button
                  onClick={() => removeClause(ruleIdx, clauseIdx)}
                  className="text-zinc-600 hover:text-red-400 transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>

          <button
            onClick={() => addClause(ruleIdx)}
            className="mt-2 flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300 transition-colors"
          >
            <Plus className="h-3 w-3" />
            Add condition
          </button>

          <div className="mt-3 flex items-center gap-4 border-t border-zinc-800 pt-3">
            <div className="flex items-center gap-2">
              <label className="text-xs text-zinc-500">Serve</label>
              <select
                value={rule.variation}
                onChange={(e) =>
                  updateRule(ruleIdx, { variation: e.target.value })
                }
                className="rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1 text-sm text-zinc-100 focus:border-blue-500/50 focus:outline-none"
              >
                {variations.map((v) => (
                  <option key={v.key} value={v.key}>
                    {v.name ?? v.key}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-zinc-500">at</label>
              <input
                type="number"
                min={0}
                max={100}
                value={rule.rolloutPercentage}
                onChange={(e) =>
                  updateRule(ruleIdx, {
                    rolloutPercentage: Number(e.target.value),
                  })
                }
                className="w-16 rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1 text-sm text-zinc-100 text-center focus:border-blue-500/50 focus:outline-none"
              />
              <span className="text-xs text-zinc-500">%</span>
            </div>
          </div>
        </div>
      ))}

      <button
        onClick={addRule}
        className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-zinc-700 py-3 text-sm text-zinc-400 hover:border-blue-500/50 hover:text-blue-400 transition-all"
      >
        <Plus className="h-4 w-4" />
        Add targeting rule
      </button>
    </div>
  );
}

export default TargetingRuleEditor;
