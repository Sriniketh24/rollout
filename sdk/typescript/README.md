# @rollout/sdk

TypeScript client SDK for the Rollout feature flag platform. Evaluates flags locally using cached rules with real-time SSE streaming updates and automatic polling fallback.

## Installation

```bash
npm install @rollout/sdk
```

## Quick Start

```typescript
import { RolloutClient } from '@rollout/sdk';

const client = new RolloutClient({
  apiKey: 'your-api-key',
  baseUrl: 'https://flags.example.com',
  environment: 'production',
});

// Wait for the client to be ready
client.on('ready', () => {
  // Evaluate a boolean flag
  const enabled = client.getBooleanValue(
    'new-checkout-flow',
    { key: 'user-123', attributes: { plan: 'pro', country: 'US' } },
    false
  );

  if (enabled) {
    showNewCheckout();
  }
});
```

## Configuration

```typescript
const client = new RolloutClient({
  apiKey: 'your-api-key',        // Required: SDK API key
  baseUrl: 'https://flags.example.com', // Required: Rollout server URL
  environment: 'production',     // Required: Environment key
  pollingInterval: 30000,        // Optional: Polling interval in ms (default: 30s)
  enableStreaming: true,          // Optional: Enable SSE streaming (default: true)
  flushInterval: 10000,           // Optional: Exposure flush interval in ms (default: 10s)
  maxBatchSize: 100,              // Optional: Max exposure events before auto-flush (default: 100)
});
```

## Evaluation Methods

```typescript
const context = {
  key: 'user-456',
  attributes: {
    email: 'alice@example.com',
    plan: 'enterprise',
    country: 'US',
    version: '2.1.0',
  },
};

// Generic evaluation
const result = client.evaluate('my-flag', context, 'default');
// result: { flag_key, value, reason, version, timestamp }

// Typed helpers
const boolVal = client.getBooleanValue('feature-x', context, false);
const strVal  = client.getStringValue('banner-text', context, 'Hello');
const numVal  = client.getNumberValue('rate-limit', context, 100);
const jsonVal = client.getJSONValue<{ color: string }>('theme', context, { color: 'blue' });

// Evaluate all flags
const allFlags = client.getAllFlags(context);
```

## Real-Time Updates

The SDK connects to the Rollout SSE stream for instant flag updates. If SSE disconnects, it automatically falls back to polling.

```typescript
client.on('flagChanged', (event) => {
  console.log(`Flag ${event.flag_key} was ${event.type}`);
});

client.on('error', (err) => {
  console.error('SDK error:', err);
});
```

## Cleanup

```typescript
await client.close();
```

This flushes any pending exposure events and shuts down streaming/polling.

## How Evaluation Works

1. On init, the SDK fetches all flag rules from the server and caches them locally.
2. Evaluations run entirely in-memory against the cached rules -- no network calls.
3. Targeting rules are evaluated in priority order; each rule's clauses must all match.
4. Percentage rollouts use SHA-256 hashing for deterministic, sticky bucketing.
5. Exposure events are batched and sent periodically for analytics.
