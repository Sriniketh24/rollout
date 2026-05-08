import { NextResponse } from "next/server";
import { fallbackDemoDataset } from "@/lib/demo-dataset";

export const dynamic = "force-dynamic";

const defaultEdgeApiUrl =
  "https://wvfosrbrbkqugrkpunhk.supabase.co/functions/v1/rollout-data";

export async function GET() {
  const upstreamUrl = process.env.ROLLOUT_EDGE_API_URL ?? defaultEdgeApiUrl;

  if (!upstreamUrl) {
    return NextResponse.json(fallbackDemoDataset);
  }

  try {
    const response = await fetch(upstreamUrl, {
      headers: { Accept: "application/json" },
      cache: "no-store",
    });

    if (!response.ok) {
      return NextResponse.json(fallbackDemoDataset);
    }

    const payload = await response.json();
    return NextResponse.json(payload);
  } catch {
    return NextResponse.json(fallbackDemoDataset);
  }
}
