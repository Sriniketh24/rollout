# rollout-sdk

Python client SDK for the Rollout feature flag platform. Evaluates flags locally using cached rules with real-time SSE streaming updates and automatic polling fallback.

## Installation

```bash
pip install rollout-sdk
```

## Quick Start

```python
from rollout_sdk import RolloutClient, EvalContext

client = RolloutClient(
    api_key="your-api-key",
    base_url="https://flags.example.com",
    environment="production",
)
client.wait_until_ready(timeout=5.0)

ctx = EvalContext(key="user-123", attributes={"plan": "pro", "country": "US"})
enabled = client.get_boolean_value("new-checkout-flow", ctx, default=False)

if enabled:
    show_new_checkout()

client.close()
```

## Context Manager

```python
with RolloutClient(api_key=..., base_url=..., environment=...) as client:
    client.wait_until_ready()
    value = client.get_boolean_value("flag", ctx, False)
```

## Configuration

```python
client = RolloutClient(
    api_key="your-api-key",           # Required: SDK API key
    base_url="https://flags.example.com",  # Required: Rollout server URL
    environment="production",         # Required: Environment key
    polling_interval=30.0,            # Optional: Polling interval in seconds (default: 30)
    enable_streaming=True,            # Optional: Enable SSE streaming (default: True)
    flush_interval=10.0,              # Optional: Exposure flush interval in seconds (default: 10)
    max_batch_size=100,               # Optional: Max exposure events before auto-flush (default: 100)
)
```

## Evaluation Methods

```python
ctx = EvalContext(
    key="user-456",
    attributes={
        "email": "alice@example.com",
        "plan": "enterprise",
        "country": "US",
    },
)

# Generic evaluation
result = client.evaluate("my-flag", ctx, default_value="fallback")
# result.flag_key, result.value, result.reason, result.version, result.timestamp

# Typed helpers
bool_val = client.get_boolean_value("feature-x", ctx, default=False)
str_val  = client.get_string_value("banner-text", ctx, default="Hello")
num_val  = client.get_number_value("rate-limit", ctx, default=100)
json_val = client.get_json_value("theme", ctx, default={"color": "blue"})

# Evaluate all flags
all_flags = client.get_all_flags(ctx)
```

## Real-Time Updates

The SDK connects via SSE for instant flag updates. If SSE disconnects, it falls back to polling automatically.

```python
def on_flag_changed(event):
    print(f"Flag {event.flag_key} was {event.type}")

client.on("flagChanged", on_flag_changed)
client.on("error", lambda e: print(f"SDK error: {e}"))
```

## How Evaluation Works

1. On init, the SDK fetches all flag rules from the server and caches them locally.
2. Evaluations run entirely in-memory against the cached rules -- no network calls.
3. Targeting rules are evaluated in priority order; each rule's clauses must all match.
4. Percentage rollouts use SHA-256 hashing for deterministic, sticky bucketing.
5. Exposure events are batched and sent periodically for analytics.
