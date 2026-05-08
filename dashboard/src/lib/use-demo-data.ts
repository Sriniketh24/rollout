"use client";

import { startTransition, useEffect, useState } from "react";
import { fallbackDemoDataset, type DemoDataset } from "@/lib/demo-dataset";

export function useDemoData() {
  const [data, setData] = useState<DemoDataset>(fallbackDemoDataset);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const response = await fetch("/api/demo-data", { cache: "no-store" });
        if (!response.ok) return;
        const nextData = (await response.json()) as DemoDataset;
        if (cancelled) return;
        startTransition(() => {
          setData(nextData);
        });
      } catch {
        // Keep the local fallback dataset if the live source is unavailable.
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  return { data, isLoading };
}
