"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import type { VariationResult } from "@/lib/types";

interface ExperimentChartProps {
  results: VariationResult[];
  metricName?: string;
}

export function ExperimentChart({
  results,
  metricName = "Conversion Rate",
}: ExperimentChartProps) {
  const data = results.map((r) => ({
    name: r.name ?? r.variationKey,
    rate: r.conversionRate,
    isControl: r.isControl,
    isWinner: r.isWinner,
    sampleSize: r.sampleSize,
  }));

  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" vertical={false} />
          <XAxis
            dataKey="name"
            tick={{ fill: "#a1a1aa", fontSize: 13 }}
            axisLine={{ stroke: "#3f3f46" }}
            tickLine={false}
          />
          <YAxis
            tick={{ fill: "#a1a1aa", fontSize: 12 }}
            axisLine={false}
            tickLine={false}
            tickFormatter={(v: number) => `${v}%`}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "#18181b",
              border: "1px solid #3f3f46",
              borderRadius: "8px",
              boxShadow: "0 10px 25px rgba(0,0,0,0.5)",
            }}
            labelStyle={{ color: "#fafafa", fontWeight: 600, marginBottom: 4 }}
            itemStyle={{ color: "#a1a1aa" }}
            formatter={(value) => {
              const numeric = typeof value === "number" ? value : Number(value ?? 0);
              return [`${numeric.toFixed(2)}%`, metricName];
            }}
          />
          <Bar dataKey="rate" radius={[6, 6, 0, 0]} barSize={60}>
            {data.map((entry, index) => (
              <Cell
                key={`cell-${index}`}
                fill={
                  entry.isWinner
                    ? "#22c55e"
                    : entry.isControl
                    ? "#3b82f6"
                    : "#8b5cf6"
                }
                fillOpacity={0.85}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

export default ExperimentChart;
