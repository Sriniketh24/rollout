"use client";

import { use } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  FlaskConical,
  TrendingUp,
  ShieldCheck,
  Lightbulb,
  CheckCircle2,
  XCircle,
  Clock,
} from "lucide-react";
import { Badge } from "@/components/Badge";
import { ExperimentChart } from "@/components/ExperimentChart";
import { mockExperiments, mockExperimentResults } from "@/lib/mock-data";

export default function ExperimentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const experiment =
    mockExperiments.find((e) => e.id === id) ?? mockExperiments[0];

  const results = mockExperimentResults;

  const statusVariant =
    experiment.status === "running"
      ? "success"
      : experiment.status === "complete"
      ? "info"
      : experiment.status === "paused"
      ? "warning"
      : experiment.status === "stopped"
      ? "danger"
      : "default";

  return (
    <div className="space-y-8">
      <Link
        href="/experiments"
        className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to experiments
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-purple-500/10">
            <FlaskConical className="h-6 w-6 text-purple-400" />
          </div>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-zinc-100">
                {experiment.name}
              </h1>
              <Badge variant={statusVariant} dot>
                {experiment.status}
              </Badge>
            </div>
            <p className="mt-0.5 font-mono text-sm text-zinc-500">
              {experiment.key}
            </p>
            <p className="mt-2 text-sm text-zinc-400">
              {experiment.description}
            </p>
          </div>
        </div>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <p className="text-xs text-zinc-500">Total Sample Size</p>
          <p className="mt-1 text-2xl font-bold text-zinc-100">
            {results.totalSampleSize.toLocaleString()}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <p className="text-xs text-zinc-500">Traffic Split</p>
          <p className="mt-1 text-2xl font-bold text-zinc-100">
            {experiment.trafficPercentage}%
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <p className="text-xs text-zinc-500">Target Metric</p>
          <p className="mt-1 text-lg font-bold text-zinc-100">
            {experiment.targetMetric.replace(/_/g, " ")}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <p className="text-xs text-zinc-500">Duration</p>
          <div className="mt-1 flex items-center gap-1.5">
            <Clock className="h-4 w-4 text-zinc-500" />
            <p className="text-lg font-bold text-zinc-100">
              {results.endDate
                ? `${Math.ceil(
                    (new Date(results.endDate).getTime() -
                      new Date(results.startDate).getTime()) /
                      86400000
                  )} days`
                : "Ongoing"}
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Chart + variation results */}
        <div className="lg:col-span-2 space-y-6">
          {/* Chart */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
            <h2 className="text-lg font-semibold text-zinc-100 mb-4">
              <TrendingUp className="inline h-5 w-5 mr-2 text-blue-400" />
              Conversion Rates
            </h2>
            <ExperimentChart
              results={results.variationResults}
              metricName="Conversion Rate"
            />
          </div>

          {/* Variation results table */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
            <h2 className="text-lg font-semibold text-zinc-100 mb-4">
              Variation Results
            </h2>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-zinc-800">
                    <th className="text-left py-3 px-2 text-zinc-500 font-medium">
                      Variation
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      Sample
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      Conversions
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      Rate
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      Improvement
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      P(beat ctrl)
                    </th>
                    <th className="text-right py-3 px-2 text-zinc-500 font-medium">
                      95% CI
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {results.variationResults.map((vr) => (
                    <tr
                      key={vr.variationKey}
                      className="border-b border-zinc-800/50"
                    >
                      <td className="py-3 px-2">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-zinc-200">
                            {vr.name ?? vr.variationKey}
                          </span>
                          {vr.isControl && (
                            <Badge variant="default">control</Badge>
                          )}
                          {vr.isWinner && (
                            <Badge variant="success">winner</Badge>
                          )}
                        </div>
                      </td>
                      <td className="text-right py-3 px-2 text-zinc-300 tabular-nums">
                        {vr.sampleSize.toLocaleString()}
                      </td>
                      <td className="text-right py-3 px-2 text-zinc-300 tabular-nums">
                        {vr.conversions.toLocaleString()}
                      </td>
                      <td className="text-right py-3 px-2 font-medium text-zinc-100 tabular-nums">
                        {vr.conversionRate.toFixed(2)}%
                      </td>
                      <td className="text-right py-3 px-2 tabular-nums">
                        {vr.isControl ? (
                          <span className="text-zinc-500">--</span>
                        ) : (
                          <span
                            className={
                              vr.improvementOverControl > 0
                                ? "text-emerald-400"
                                : "text-red-400"
                            }
                          >
                            {vr.improvementOverControl > 0 ? "+" : ""}
                            {vr.improvementOverControl.toFixed(2)}%
                          </span>
                        )}
                      </td>
                      <td className="text-right py-3 px-2 tabular-nums">
                        {vr.isControl ? (
                          <span className="text-zinc-500">--</span>
                        ) : (
                          <span
                            className={
                              vr.probabilityToBeatControl > 95
                                ? "text-emerald-400 font-medium"
                                : "text-zinc-300"
                            }
                          >
                            {vr.probabilityToBeatControl.toFixed(1)}%
                          </span>
                        )}
                      </td>
                      <td className="text-right py-3 px-2 text-zinc-400 tabular-nums text-xs">
                        [{vr.credibleInterval.lower.toFixed(2)}%,{" "}
                        {vr.credibleInterval.upper.toFixed(2)}%]
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Bayesian recommendation */}
          {results.recommendation && (
            <div
              className={`rounded-xl border p-6 ${
                results.recommendation.action === "ship"
                  ? "border-emerald-500/30 bg-emerald-500/5"
                  : results.recommendation.action === "stop"
                  ? "border-red-500/30 bg-red-500/5"
                  : "border-yellow-500/30 bg-yellow-500/5"
              }`}
            >
              <div className="flex items-center gap-2 mb-3">
                <Lightbulb
                  className={`h-5 w-5 ${
                    results.recommendation.action === "ship"
                      ? "text-emerald-400"
                      : results.recommendation.action === "stop"
                      ? "text-red-400"
                      : "text-yellow-400"
                  }`}
                />
                <h3 className="text-sm font-semibold text-zinc-100">
                  AI Recommendation
                </h3>
              </div>
              <div className="flex items-center gap-2 mb-3">
                <Badge
                  variant={
                    results.recommendation.action === "ship"
                      ? "success"
                      : results.recommendation.action === "stop"
                      ? "danger"
                      : "warning"
                  }
                >
                  {results.recommendation.action.toUpperCase()}
                </Badge>
                <span className="text-xs text-zinc-400">
                  {(results.recommendation.confidence * 100).toFixed(0)}%
                  confidence
                </span>
              </div>
              <p className="text-sm text-zinc-300 leading-relaxed">
                {results.recommendation.reasoning}
              </p>
              <div className="mt-4 space-y-2">
                <p className="text-xs font-medium text-zinc-400 uppercase tracking-wider">
                  Next Steps
                </p>
                {results.recommendation.suggestedNextSteps.map((step, i) => (
                  <div key={i} className="flex items-start gap-2">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 mt-0.5 shrink-0" />
                    <p className="text-xs text-zinc-400">{step}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Guardrail metrics */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
            <div className="flex items-center gap-2 mb-4">
              <ShieldCheck className="h-5 w-5 text-blue-400" />
              <h3 className="text-sm font-semibold text-zinc-100">
                Guardrail Metrics
              </h3>
            </div>
            {results.guardrailResults.length > 0 ? (
              <div className="space-y-3">
                {results.guardrailResults.map((gr) => (
                  <div
                    key={gr.metricKey}
                    className="rounded-lg border border-zinc-800 bg-zinc-950 p-3"
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium text-zinc-200">
                        {gr.metricName}
                      </span>
                      {gr.passed ? (
                        <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-500" />
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-xs text-zinc-400">
                      <span>Control: {gr.controlValue}</span>
                      <span>Treatment: {gr.treatmentValue}</span>
                    </div>
                    <p className="mt-1 text-xs text-zinc-500">{gr.message}</p>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-zinc-500">
                No guardrail metrics configured
              </p>
            )}
          </div>

          {/* Experiment info */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
            <h3 className="text-sm font-semibold text-zinc-100 mb-3">
              Details
            </h3>
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between">
                <dt className="text-zinc-500">Flag</dt>
                <dd>
                  <Link
                    href={`/flags/${experiment.flagId}`}
                    className="text-blue-400 hover:text-blue-300"
                  >
                    {experiment.flagKey ?? experiment.flagId}
                  </Link>
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-zinc-500">Variations</dt>
                <dd className="text-zinc-300">
                  {experiment.variations.length}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-zinc-500">Traffic</dt>
                <dd className="text-zinc-300">
                  {experiment.trafficPercentage}%
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-zinc-500">Created</dt>
                <dd className="text-zinc-300">
                  {new Date(experiment.createdAt).toLocaleDateString()}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>
    </div>
  );
}
