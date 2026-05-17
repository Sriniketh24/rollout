# Rollout

**A feature flag and experimentation platform built for resume-grade systems work.**

Rollout is a LaunchDarkly/Optimizely-style system for safely shipping features, running experiments, and analyzing impact. The live free deployment path uses `Vercel Hobby + Supabase Free`: visitors can inspect a public demo workspace immediately, start an email-free editable guest workspace, or sign in for an isolated workspace with persistent flags and environments. The repo also contains a deeper Go-based control plane and local multi-service architecture for systems-focused iteration.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           ROLLOUT ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌──────────┐    ┌──────────────────────────────────────────────┐      │
│   │Dashboard │───▶│        Vercel App / Demo Data Proxy          │      │
│   │(Next.js) │    │  UI · route handlers · deployable frontend   │      │
│   └──────────┘    └──────────────────┬───────────────────────────┘      │
│                                      │                                  │
│                                      ▼                                  │
│                           ┌──────────────────────┐                      │
│                           │ Supabase Postgres    │                      │
│                           │ schema · audit · exps│                      │
│                           └──────────┬───────────┘                      │
│                                      │                                  │
│                                      ▼                                  │
│                           ┌──────────────────────┐                      │
│                           │ Supabase Edge Func   │                      │
│                           │ read-only demo API   │                      │
│                           └──────────────────────┘                      │
│                                                                         │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │ Local / next-phase architecture in repo: Go control plane, edge │  │
│   │ relay, Redis/NATS/ClickHouse integrations, SDKs, load tests     │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Performance

Benchmarked on Apple M1:

| Metric | Result |
|--------|--------|
| Simple flag evaluation | **137 ns** (0 allocs) |
| Evaluation with targeting rules | **314 ns** (2 allocs) |
| Rollout bucketing (SHA-256) | **387 ns** (2 allocs) |
| p95 flag evaluation | **< 1 μs** |
| Config propagation (SSE) | **< 250 ms** |

All evaluations happen locally in-process — no network calls in the hot path.

## Features

### Core Platform
- **Feature Flags** — Boolean, string, number, and JSON flag types with per-environment configuration
- **Targeting Rules** — Attribute-based targeting with 12+ operators (equals, contains, regex, semver, etc.)
- **Percentage Rollouts** — Deterministic sticky bucketing via SHA-256 hashing for consistent user assignment
- **Kill Switches** — Emergency flag disable that overrides all rules instantly
- **Flag Dependencies** — DAG-based dependency resolution between flags with automatic propagation
- **Multi-Environment** — Separate configurations per environment (dev, staging, production) with audit trails

### Experimentation
- **A/B Testing** — Exposure tracking, conversion events, holdout groups
- **Bayesian Analysis** — Posterior sampling with probability-to-beat-control, expected loss, and 95% credible intervals
- **Guardrail Metrics** — Automatic degradation detection on latency, error rates, and custom metrics
- **AI-Powered Insights** — Automated experiment recommendations (Ship / Investigate / Continue)

### Infrastructure
- **Supabase-backed demo deployment** — Strictly free hosted path using Supabase Postgres plus an Edge Function-backed demo API
- **Editable guest workspaces** — Email-free browser-token workspaces let recruiters create and update real persisted flags without signup friction
- **Authenticated workspaces** — Supabase Auth provisions private projects with default environments and RLS-isolated data
- **Edge Relay** — Lightweight Go binary that caches rules locally and evaluates flags at the edge (< 5ms p99)
- **Streaming Updates** — Server-Sent Events (SSE) for real-time config propagation to SDKs and relays
- **Event Ingestion Pipeline** — Buffered batch inserts into ClickHouse via NATS for the local/distributed architecture track
- **RBAC** — Role-based access control (Admin, Editor, Viewer) with JWT + API key authentication
- **Audit Logging** — Complete history of every flag change, rollout, and experiment action
- **Chaos Engineering** — Built-in fault injection: failure simulation, latency injection, stale cache emulation
- **Health Checks** — Per-component health monitoring with degraded state detection

### Developer Experience
- **GitOps** — Declare flag state in `rollout.yaml`, sync via CLI or GitHub Action on merge
- **CLI Tool** — Full flag management, experiment results, audit logs, and health checks from the terminal
- **TypeScript SDK** — Local evaluation, SSE streaming, exposure batching, event emitter
- **Python SDK** — Thread-safe client with context manager support, background streaming
- **REST API** — Complete CRUD with OpenAPI-compatible endpoints

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.23 (net/http, structured logging) |
| Dashboard | Next.js 16, TypeScript, Tailwind CSS, Recharts |
| Hosted Data Plane | Supabase Postgres + Supabase Edge Functions |
| Local Infra Track | PostgreSQL, Redis, NATS, ClickHouse |
| Infrastructure | Docker Compose, GitHub Actions |
| Deployment | Vercel Hobby + Supabase Free |

## Project Structure

```
rollout/
├── cmd/
│   ├── rollout-server/     # Control plane API server
│   ├── rollout-relay/      # Edge relay for local flag evaluation
│   ├── rollout-ingest/     # Event ingestion worker (NATS → ClickHouse)
│   └── rollout-cli/        # CLI tool for GitOps and management
├── internal/
│   ├── analytics/          # ClickHouse queries and event ingestion
│   ├── audit/              # Audit logging
│   ├── auth/               # JWT + API key authentication, RBAC
│   ├── cache/              # Redis cache layer with TTL management
│   ├── chaos/              # Fault injection engine
│   ├── config/             # Environment-based configuration
│   ├── events/             # NATS event bus
│   ├── experiment/         # Bayesian analysis engine
│   ├── flag/               # Flag evaluation engine (sub-μs)
│   ├── health/             # Health check framework
│   ├── middleware/         # HTTP middleware (auth, CORS, rate limit, logging)
│   ├── models/             # Domain models
│   ├── store/              # PostgreSQL data access layer
│   ├── streaming/          # SSE hub for real-time updates
│   └── targeting/          # Targeting rule engine with regex caching
├── dashboard/              # Next.js admin UI
├── sdk/
│   ├── typescript/         # TypeScript SDK with local evaluation
│   └── python/             # Python SDK with threading support
├── supabase/
│   ├── functions/          # Supabase Edge Function used by the free hosted demo
│   ├── migrations/         # Hosted Postgres schema for the free deployment
│   └── seed.sql            # Deterministic demo dataset for the hosted path
├── migrations/             # PostgreSQL + ClickHouse schemas
├── tests/load/             # Load testing suite
├── deployments/            # Docker configs for the local/distributed stack
├── .github/workflows/      # CI pipeline + GitOps sync
├── rollout.yaml            # Example GitOps flag configuration
└── docker-compose.yml      # Full local development stack
```

## Quick Start

### Prerequisites
- Go 1.23+
- Node.js 20+
- Docker & Docker Compose

### Free hosted deployment

- Dashboard: Vercel Hobby
- Data and demo API: Supabase Free
- Current Supabase demo function: `https://wvfosrbrbkqugrkpunhk.supabase.co/functions/v1/rollout-data`
- Production dashboard: `https://dashboard-rho-roan.vercel.app`
- Public visitors see a neutral read-only demo workspace.
- Recruiters can start an editable guest workspace without email confirmation or account creation.
- Signed-in users get their own workspace, default environments, persistent flag creation, environment creation, rollout updates, kill-switch updates, targeting-rule updates, and audit logs.

### Rebuild the hosted demo data

```bash
# Apply the Supabase schema tracked in this repo
supabase db push

# Seed the deterministic demo project
psql "$SUPABASE_DB_URL" -f supabase/seed.sql
```

### Run locally

```bash
# Clone the repo
git clone https://github.com/Sriniketh24/rollout.git
cd rollout

# Start all services
docker compose up -d

# For the local distributed stack:
#   Control Plane API:  http://localhost:8080
#   Edge Relay:         http://localhost:8081
#   Dashboard:          http://localhost:3000
#   NATS Monitor:       http://localhost:8222
```

### Using the CLI

```bash
# Build the CLI
go build -o rollout-cli ./cmd/rollout-cli

# Authenticate
./rollout-cli login --api-key your-api-key --url http://localhost:8080

# Create a flag
./rollout-cli flags create dark-mode --name "Dark Mode" --type boolean

# Toggle it on
./rollout-cli flags toggle dark-mode --env production

# GitOps sync
./rollout-cli sync --file rollout.yaml

# Check experiment results
./rollout-cli experiments results <experiment-id>
```

### Using the TypeScript SDK

```typescript
import { RolloutClient } from '@rollout/sdk';

const client = new RolloutClient({
  apiKey: 'rol_your_api_key',
  baseUrl: 'http://localhost:8080',
  environment: 'production',
  enableStreaming: true,
});

client.on('ready', async () => {
  const darkMode = await client.getBooleanValue('dark-mode', {
    key: 'user-123',
    attributes: { plan: 'premium', country: 'US' },
  }, false);
  
  console.log('Dark mode:', darkMode);
});
```

### Using the Python SDK

```python
from rollout_sdk import RolloutClient, EvalContext

with RolloutClient(
    api_key="rol_your_api_key",
    base_url="http://localhost:8080",
    environment="production",
) as client:
    client.wait_until_ready()
    
    dark_mode = client.get_boolean_value(
        "dark-mode",
        EvalContext(key="user-123", attributes={"plan": "premium"}),
        default=False,
    )
    print(f"Dark mode: {dark_mode}")
```

## Running Tests

```bash
# Unit tests
go test -v ./...

# Benchmarks
go test -bench=. -benchmem ./internal/flag/ ./internal/targeting/ ./internal/experiment/

# Load tests (requires running server)
go test -v -run TestLoad ./tests/load/
```

## Design Decisions

### Why local evaluation?
Flag evaluation happens entirely in-process using cached rules — no network round-trip. This achieves sub-microsecond latency and eliminates the evaluation endpoint as a single point of failure. Rules are synced via SSE streaming with polling fallback.

### Why Bayesian over frequentist A/B testing?
Bayesian analysis provides intuitive metrics (probability to beat control, expected loss) that directly answer business questions without requiring fixed sample sizes. It also supports continuous monitoring — you can check results at any time without inflating false positive rates.

### Why SHA-256 for bucketing?
Deterministic hashing ensures users always see the same variation (sticky bucketing) regardless of which server or edge relay evaluates the flag. SHA-256 provides excellent distribution uniformity verified by automated tests.

### Why an edge relay?
The relay enables sub-5ms flag evaluation at the network edge, reduces load on the control plane, and provides resilience — SDKs continue working with cached rules if the control plane is temporarily unavailable.

## License

MIT
