import {
  mockAuditEntries,
  mockEnvironments,
  mockExperimentResults,
  mockExperiments,
  mockFlags,
} from "@/lib/mock-data";
import type {
  AuditEntry,
  Environment,
  Experiment,
  ExperimentResults,
  Flag,
  Project,
} from "@/lib/types";

export interface DemoDataset {
  project: Project;
  environments: Environment[];
  flags: Flag[];
  experiments: Experiment[];
  experimentResultsById: Record<string, ExperimentResults>;
  auditEntries: AuditEntry[];
}

export const fallbackDemoDataset: DemoDataset = {
  project: {
    id: "proj-1",
    key: "rollout-demo",
    name: "Rollout Demo",
    description: "Fallback local dataset",
    apiKey: "rol_demo_owner_key",
    serverApiKey: "rol_demo_server_key",
    createdAt: "2025-01-10T08:00:00Z",
    updatedAt: "2025-05-07T10:00:00Z",
  },
  environments: mockEnvironments,
  flags: mockFlags,
  experiments: mockExperiments,
  experimentResultsById: {
    "exp-2": mockExperimentResults,
    "exp-1": {
      ...mockExperimentResults,
      experimentId: "exp-1",
      status: "running",
      endDate: undefined,
      totalSampleSize: 182430,
      lastUpdated: "2025-05-07T12:00:00Z",
      recommendation: {
        action: "ship",
        confidence: 0.91,
        reasoning:
          "Variant B is ahead with strong probability and no meaningful guardrail regression.",
        suggestedNextSteps: [
          "Promote the streamlined checkout to 100% of production traffic",
          "Watch latency for 24 hours after rollout",
        ],
      },
    },
  },
  auditEntries: mockAuditEntries,
};
