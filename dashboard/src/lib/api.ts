import type {
  Flag,
  Environment,
  Experiment,
  ExperimentResults,
  AuditEntry,
  User,
  Project,
  EvalContext,
  EvalResult,
  PaginatedResponse,
  SSEEvent,
} from "./types";

// ── Configuration ──────────────────────────────────────────────────

const BASE_URL =
  typeof window !== "undefined"
    ? (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080")
    : (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080");

// ── HTTP helpers ───────────────────────────────────────────────────

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiClientError(
      res.status,
      body.message ?? res.statusText,
      body.code,
      body.details
    );
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export class ApiClientError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string,
    public details?: Record<string, string>
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

// ── Flags ──────────────────────────────────────────────────────────

export interface CreateFlagPayload {
  key: string;
  name: string;
  description?: string;
  type: Flag["type"];
  tags?: string[];
  defaultVariation?: string;
  variations?: Flag["variations"];
}

export interface UpdateFlagPayload {
  name?: string;
  description?: string;
  tags?: string[];
  archived?: boolean;
  defaultVariation?: string;
  variations?: Flag["variations"];
}

export interface ToggleFlagPayload {
  environmentId: string;
  enabled: boolean;
}

export interface UpdateRolloutPayload {
  environmentId: string;
  rolloutPercentage: number;
}

export interface UpdateTargetingPayload {
  environmentId: string;
  rules: Flag["environments"][string]["targetingRules"];
}

export const flagsApi = {
  list(params?: {
    page?: number;
    pageSize?: number;
    search?: string;
    tag?: string;
    type?: string;
    archived?: boolean;
  }): Promise<PaginatedResponse<Flag>> {
    const q = new URLSearchParams();
    if (params?.page) q.set("page", String(params.page));
    if (params?.pageSize) q.set("pageSize", String(params.pageSize));
    if (params?.search) q.set("search", params.search);
    if (params?.tag) q.set("tag", params.tag);
    if (params?.type) q.set("type", params.type);
    if (params?.archived !== undefined)
      q.set("archived", String(params.archived));
    const qs = q.toString();
    return request<PaginatedResponse<Flag>>(`/api/flags${qs ? `?${qs}` : ""}`);
  },

  get(id: string): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}`);
  },

  create(payload: CreateFlagPayload): Promise<Flag> {
    return request<Flag>("/api/flags", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  update(id: string, payload: UpdateFlagPayload): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/flags/${id}`, { method: "DELETE" });
  },

  toggle(id: string, payload: ToggleFlagPayload): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}/toggle`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  updateRollout(id: string, payload: UpdateRolloutPayload): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}/rollout`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  updateTargeting(id: string, payload: UpdateTargetingPayload): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}/targeting`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  killSwitch(
    id: string,
    environmentId: string,
    active: boolean
  ): Promise<Flag> {
    return request<Flag>(`/api/flags/${id}/kill`, {
      method: "POST",
      body: JSON.stringify({ environmentId, active }),
    });
  },

  evaluate(flagKey: string, context: EvalContext): Promise<EvalResult> {
    return request<EvalResult>(`/api/flags/${flagKey}/evaluate`, {
      method: "POST",
      body: JSON.stringify(context),
    });
  },
};

// ── Experiments ────────────────────────────────────────────────────

export interface CreateExperimentPayload {
  key: string;
  name: string;
  description?: string;
  flagId: string;
  variations: Experiment["variations"];
  trafficPercentage: number;
  targetMetric: string;
  guardrailMetrics?: Experiment["guardrailMetrics"];
}

export interface UpdateExperimentPayload {
  name?: string;
  description?: string;
  trafficPercentage?: number;
  guardrailMetrics?: Experiment["guardrailMetrics"];
}

export const experimentsApi = {
  list(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
  }): Promise<PaginatedResponse<Experiment>> {
    const q = new URLSearchParams();
    if (params?.page) q.set("page", String(params.page));
    if (params?.pageSize) q.set("pageSize", String(params.pageSize));
    if (params?.status) q.set("status", params.status);
    const qs = q.toString();
    return request<PaginatedResponse<Experiment>>(
      `/api/experiments${qs ? `?${qs}` : ""}`
    );
  },

  get(id: string): Promise<Experiment> {
    return request<Experiment>(`/api/experiments/${id}`);
  },

  create(payload: CreateExperimentPayload): Promise<Experiment> {
    return request<Experiment>("/api/experiments", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  update(id: string, payload: UpdateExperimentPayload): Promise<Experiment> {
    return request<Experiment>(`/api/experiments/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/experiments/${id}`, { method: "DELETE" });
  },

  start(id: string): Promise<Experiment> {
    return request<Experiment>(`/api/experiments/${id}/start`, {
      method: "POST",
    });
  },

  pause(id: string): Promise<Experiment> {
    return request<Experiment>(`/api/experiments/${id}/pause`, {
      method: "POST",
    });
  },

  stop(id: string): Promise<Experiment> {
    return request<Experiment>(`/api/experiments/${id}/stop`, {
      method: "POST",
    });
  },

  results(id: string): Promise<ExperimentResults> {
    return request<ExperimentResults>(`/api/experiments/${id}/results`);
  },
};

// ── Environments ───────────────────────────────────────────────────

export interface CreateEnvironmentPayload {
  key: string;
  name: string;
  color: string;
  description?: string;
  production?: boolean;
}

export interface UpdateEnvironmentPayload {
  name?: string;
  color?: string;
  description?: string;
  production?: boolean;
  order?: number;
}

export const environmentsApi = {
  list(): Promise<Environment[]> {
    return request<Environment[]>("/api/environments");
  },

  get(id: string): Promise<Environment> {
    return request<Environment>(`/api/environments/${id}`);
  },

  create(payload: CreateEnvironmentPayload): Promise<Environment> {
    return request<Environment>("/api/environments", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  update(id: string, payload: UpdateEnvironmentPayload): Promise<Environment> {
    return request<Environment>(`/api/environments/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
  },

  delete(id: string): Promise<void> {
    return request<void>(`/api/environments/${id}`, { method: "DELETE" });
  },
};

// ── Users ──────────────────────────────────────────────────────────

export interface InviteUserPayload {
  email: string;
  role: User["role"];
}

export const usersApi = {
  list(): Promise<User[]> {
    return request<User[]>("/api/users");
  },

  get(id: string): Promise<User> {
    return request<User>(`/api/users/${id}`);
  },

  invite(payload: InviteUserPayload): Promise<User> {
    return request<User>("/api/users/invite", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  updateRole(id: string, role: User["role"]): Promise<User> {
    return request<User>(`/api/users/${id}/role`, {
      method: "PATCH",
      body: JSON.stringify({ role }),
    });
  },

  remove(id: string): Promise<void> {
    return request<void>(`/api/users/${id}`, { method: "DELETE" });
  },
};

// ── Projects ───────────────────────────────────────────────────────

export interface UpdateProjectPayload {
  name?: string;
  description?: string;
}

export const projectsApi = {
  list(): Promise<Project[]> {
    return request<Project[]>("/api/projects");
  },

  get(id: string): Promise<Project> {
    return request<Project>(`/api/projects/${id}`);
  },

  create(payload: { key: string; name: string; description?: string }): Promise<Project> {
    return request<Project>("/api/projects", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  update(id: string, payload: UpdateProjectPayload): Promise<Project> {
    return request<Project>(`/api/projects/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
  },

  regenerateKey(id: string): Promise<{ apiKey: string }> {
    return request<{ apiKey: string }>(`/api/projects/${id}/regenerate-key`, {
      method: "POST",
    });
  },
};

// ── Audit Log ──────────────────────────────────────────────────────

export const auditApi = {
  list(params?: {
    page?: number;
    pageSize?: number;
    action?: string;
    actor?: string;
    from?: string;
    to?: string;
    search?: string;
  }): Promise<PaginatedResponse<AuditEntry>> {
    const q = new URLSearchParams();
    if (params?.page) q.set("page", String(params.page));
    if (params?.pageSize) q.set("pageSize", String(params.pageSize));
    if (params?.action) q.set("action", params.action);
    if (params?.actor) q.set("actor", params.actor);
    if (params?.from) q.set("from", params.from);
    if (params?.to) q.set("to", params.to);
    if (params?.search) q.set("search", params.search);
    const qs = q.toString();
    return request<PaginatedResponse<AuditEntry>>(
      `/api/audit${qs ? `?${qs}` : ""}`
    );
  },
};

// ── Dashboard Stats ────────────────────────────────────────────────

export interface DashboardStats {
  totalFlags: number;
  activeExperiments: number;
  environments: number;
  recentEvents: number;
}

export const dashboardApi = {
  stats(): Promise<DashboardStats> {
    return request<DashboardStats>("/api/dashboard/stats");
  },
};

// ── SSE (Server-Sent Events) ───────────────────────────────────────

export function subscribeToUpdates(
  onEvent: (event: SSEEvent) => void,
  onError?: (error: Event) => void
): () => void {
  const url = `${BASE_URL}/api/stream`;
  const eventSource = new EventSource(url);

  eventSource.onmessage = (e) => {
    try {
      const event = JSON.parse(e.data) as SSEEvent;
      onEvent(event);
    } catch {
      // ignore malformed events
    }
  };

  eventSource.onerror = (e) => {
    onError?.(e);
  };

  return () => {
    eventSource.close();
  };
}
