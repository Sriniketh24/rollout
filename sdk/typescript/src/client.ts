import { Evaluator, createFlagStore } from './evaluator';
import type {
  EvalContext,
  EvalResult,
  ExposureEvent,
  Flag,
  FlagEnvironment,
  FlagRulesResponse,
  RolloutClientConfig,
  SSEEvent,
} from './types';

// ─── Event Emitter ───────────────────────────────────────────────────────────

type EventMap = {
  flagChanged: (event: SSEEvent) => void;
  ready: () => void;
  error: (error: Error) => void;
};

type EventName = keyof EventMap;

// ─── RolloutClient ───────────────────────────────────────────────────────────

export class RolloutClient {
  private readonly config: Required<RolloutClientConfig>;
  private readonly evaluator: Evaluator;
  private readonly flagStore = createFlagStore();
  private readonly listeners: Map<EventName, Set<EventMap[EventName]>> = new Map();
  private readonly exposureBuffer: ExposureEvent[] = [];
  private readonly seenExposures: Set<string> = new Set();

  private pollingTimer: ReturnType<typeof setInterval> | null = null;
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private eventSource: EventSource | null = null;
  private ready = false;
  private closed = false;

  constructor(config: RolloutClientConfig) {
    this.config = {
      apiKey: config.apiKey,
      baseUrl: config.baseUrl.replace(/\/+$/, ''),
      environment: config.environment,
      pollingInterval: config.pollingInterval ?? 30_000,
      enableStreaming: config.enableStreaming ?? true,
      flushInterval: config.flushInterval ?? 10_000,
      maxBatchSize: config.maxBatchSize ?? 100,
    };

    this.evaluator = new Evaluator(this.flagStore);

    // Fire-and-forget initialisation — callers can await init() or listen for 'ready'.
    this.init().catch((err) => this.emit('error', err as Error));
  }

  // ─── Initialisation ──────────────────────────────────────────────────────

  /** Fetch initial flag rules, start streaming/polling, and start the flush timer. */
  async init(): Promise<void> {
    await this.fetchFlagRules();
    this.ready = true;
    this.emit('ready');

    if (this.config.enableStreaming) {
      this.startStreaming();
    } else {
      this.startPolling();
    }

    this.flushTimer = setInterval(() => {
      this.flushExposures().catch(() => {});
    }, this.config.flushInterval);
  }

  // ─── Evaluation API ──────────────────────────────────────────────────────

  /**
   * Evaluate a flag locally using cached rules.
   * Returns an {@link EvalResult} with the value, reason, and metadata.
   */
  evaluate(flagKey: string, context: EvalContext, defaultValue: unknown = null): EvalResult {
    const result = this.evaluator.evaluate(flagKey, context, defaultValue);
    this.trackExposure(flagKey, context, result);
    return result;
  }

  /** Convenience: evaluate a flag and return a boolean. */
  getBooleanValue(flagKey: string, context: EvalContext, defaultValue = false): boolean {
    const result = this.evaluate(flagKey, context, defaultValue);
    return typeof result.value === 'boolean' ? result.value : defaultValue;
  }

  /** Convenience: evaluate a flag and return a string. */
  getStringValue(flagKey: string, context: EvalContext, defaultValue = ''): string {
    const result = this.evaluate(flagKey, context, defaultValue);
    return typeof result.value === 'string' ? result.value : defaultValue;
  }

  /** Convenience: evaluate a flag and return a number. */
  getNumberValue(flagKey: string, context: EvalContext, defaultValue = 0): number {
    const result = this.evaluate(flagKey, context, defaultValue);
    return typeof result.value === 'number' ? result.value : defaultValue;
  }

  /** Convenience: evaluate a flag and return an arbitrary JSON value. */
  getJSONValue<T = unknown>(flagKey: string, context: EvalContext, defaultValue: T | null = null): T {
    const result = this.evaluate(flagKey, context, defaultValue);
    return result.value as T;
  }

  /** Evaluate all cached flags for the given context. */
  getAllFlags(context: EvalContext): EvalResult[] {
    const results = this.evaluator.evaluateAll(context);
    for (const result of results) {
      this.trackExposure(result.flag_key, context, result);
    }
    return results;
  }

  /** Returns `true` once the initial flag rules have been fetched. */
  isReady(): boolean {
    return this.ready;
  }

  // ─── Event Emitter ─────────────────────────────────────────────────────

  on<E extends EventName>(event: E, callback: EventMap[E]): this {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
    return this;
  }

  off<E extends EventName>(event: E, callback: EventMap[E]): this {
    this.listeners.get(event)?.delete(callback);
    return this;
  }

  private emit<E extends EventName>(event: E, ...args: Parameters<EventMap[E]>): void {
    const callbacks = this.listeners.get(event);
    if (!callbacks) return;
    for (const cb of callbacks) {
      try {
        (cb as (...a: unknown[]) => void)(...args);
      } catch {
        // Listener errors must not crash the SDK.
      }
    }
  }

  // ─── Flag Fetching ─────────────────────────────────────────────────────

  private async fetchFlagRules(): Promise<void> {
    const url = `${this.config.baseUrl}/v1/sdk/flags?environment=${encodeURIComponent(this.config.environment)}`;
    const res = await fetch(url, {
      headers: {
        Authorization: `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json',
      },
    });

    if (!res.ok) {
      throw new Error(`Failed to fetch flag rules: ${res.status} ${res.statusText}`);
    }

    const data: FlagRulesResponse = await res.json();
    this.updateStore(data.flags, data.flag_environments);
  }

  private updateStore(flags: Flag[], flagEnvironments: FlagEnvironment[]): void {
    this.flagStore.flags.clear();
    for (const flag of flags) {
      this.flagStore.flags.set(flag.key, flag);
    }
    this.flagStore.flagEnvironments.clear();
    for (const fe of flagEnvironments) {
      this.flagStore.flagEnvironments.set(fe.flag_id, fe);
    }
  }

  // ─── SSE Streaming ─────────────────────────────────────────────────────

  private startStreaming(): void {
    if (this.closed) return;

    const url = `${this.config.baseUrl}/v1/stream?env_id=${encodeURIComponent(this.config.environment)}`;

    // EventSource is available in modern browsers and Node 18+ (undici).
    // In environments where it is absent, fall back to polling.
    if (typeof EventSource === 'undefined') {
      this.startPolling();
      return;
    }

    try {
      this.eventSource = new EventSource(url);

      // Listen for typed flag events
      const flagEventTypes = [
        'flag.updated',
        'flag.created',
        'flag.deleted',
        'flag.toggled',
      ];

      for (const eventType of flagEventTypes) {
        this.eventSource.addEventListener(eventType, (event: MessageEvent) => {
          try {
            const sseEvent: SSEEvent = JSON.parse(event.data);
            this.handleSSEEvent(sseEvent);
          } catch {
            // Ignore malformed events.
          }
        });
      }

      this.eventSource.onerror = () => {
        // On error, close SSE and fall back to polling.
        this.eventSource?.close();
        this.eventSource = null;
        this.startPolling();
      };
    } catch {
      this.startPolling();
    }
  }

  private handleSSEEvent(event: SSEEvent): void {
    // Re-fetch all flag rules to stay in sync.
    this.fetchFlagRules().catch(() => {});
    this.emit('flagChanged', event);
  }

  // ─── Polling Fallback ──────────────────────────────────────────────────

  private startPolling(): void {
    if (this.closed || this.pollingTimer) return;

    this.pollingTimer = setInterval(() => {
      this.fetchFlagRules().catch(() => {});
    }, this.config.pollingInterval);
  }

  private stopPolling(): void {
    if (this.pollingTimer) {
      clearInterval(this.pollingTimer);
      this.pollingTimer = null;
    }
  }

  // ─── Exposure Tracking ─────────────────────────────────────────────────

  private trackExposure(flagKey: string, context: EvalContext, result: EvalResult): void {
    // Deduplicate: only track each (flagKey, userKey) combination once per session.
    const dedupeKey = `${flagKey}:${context.key}`;
    if (this.seenExposures.has(dedupeKey)) return;
    this.seenExposures.add(dedupeKey);

    const event: ExposureEvent = {
      flag_key: flagKey,
      environment_id: this.config.environment,
      user_key: context.key,
      variation: result.value,
      reason: result.reason,
      timestamp: new Date().toISOString(),
    };

    this.exposureBuffer.push(event);

    if (this.exposureBuffer.length >= this.config.maxBatchSize) {
      this.flushExposures().catch(() => {});
    }
  }

  private async flushExposures(): Promise<void> {
    if (this.exposureBuffer.length === 0) return;

    const batch = this.exposureBuffer.splice(0, this.exposureBuffer.length);

    try {
      const url = `${this.config.baseUrl}/v1/sdk/exposures`;
      await fetch(url, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.config.apiKey}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ events: batch }),
      });
    } catch {
      // On failure, put events back so they can be retried on the next flush.
      this.exposureBuffer.unshift(...batch);
    }
  }

  // ─── Cleanup ───────────────────────────────────────────────────────────

  /** Flush pending exposure events and shut down all background tasks. */
  async close(): Promise<void> {
    this.closed = true;

    this.stopPolling();

    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    // Best-effort final flush.
    await this.flushExposures().catch(() => {});

    this.listeners.clear();
  }
}
