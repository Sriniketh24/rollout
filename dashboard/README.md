# Rollout Dashboard

Next.js App Router dashboard for the Rollout control plane.

## Development

```bash
npm install
npm run dev
```

The dashboard runs at http://localhost:3000 and expects the control plane API at `NEXT_PUBLIC_API_URL`, defaulting to http://localhost:8080.

## Verification

```bash
npm run lint
npx tsc --noEmit
npm run build
```

The production build uses `output: "standalone"` so the Docker image can run with `node server.js`.
