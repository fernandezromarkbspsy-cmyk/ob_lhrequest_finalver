# Setup Guide

This project implements the truck request portal guide with a free-first stack: Go + Chi API, Supabase Postgres, React + TypeScript frontend, optional Docker/NGINX, GitHub Actions, and optional free-tier integrations.

## Local Requirements

- Go 1.25+
- Node.js 22+ for `frontend-react`
- Docker Desktop, optional for containerized runs
- A Supabase Postgres project
- GitHub account for CI/CD
- Optional accounts: Clerk, Resend, PostHog, Sentry, Upstash, Pinecone, Better Stack, Cloudflare, Namecheap, Vercel

## Environment Setup

Copy the example env file:

```powershell
Copy-Item .env.example .env
```

Set at minimum:

```env
APP_ENV=development
APP_HOST=127.0.0.1
APP_PORT=8080
FRONTEND_URL=http://localhost:5173
JWT_SECRET=replace-with-a-long-random-secret
DATABASE_URL=postgres://user:password@db.project-ref.supabase.co:5432/postgres?sslmode=require
RATE_LIMIT_PER_MINUTE=120
FRONTEND_DIR=frontend-react/dist
FRONTEND_API_URL=http://127.0.0.1:8080
```

Optional integrations stay disabled until their keys are present:

```env
CLERK_SECRET_KEY=
RESEND_API_KEY=
POSTHOG_API_KEY=
SENTRY_DSN=
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
PINECONE_API_KEY=
BETTER_STACK_SOURCE_TOKEN=
```

## Run App Locally

Backend:

```powershell
go run .\cmd\server
```

React frontend:

```powershell
Set-Location frontend-react
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

The Vite proxy sends `/api` and `/healthz` requests to `http://127.0.0.1:8080`.

To serve the production React build through Go:

```powershell
Set-Location frontend-react
npm run build
Set-Location ..
go run .\cmd\frontend
```

`cmd/frontend` serves `frontend-react/dist` and proxies `/api` and `/healthz` to `FRONTEND_API_URL`.

## Docker + NGINX

Docker is optional. If Docker Desktop is available, run API, React frontend build, and NGINX reverse proxy:

```powershell
docker compose up --build
```

Open the proxied app:

```text
http://localhost:8088
```

NGINX routes:

- `/api/*` -> Go API
- `/healthz` -> Go API
- `/` -> frontend

## Backend Architecture

Active API groups are organized by feature:

```text
internal/features/auth/
internal/features/requests/
internal/features/users/
internal/features/clusters/
internal/features/notifications/
internal/features/realtime/
```

Each feature follows:

```text
controller.go  -> HTTP request/response only
service.go     -> business rules and workflow logic
repository.go  -> database queries only
types.go       -> DTOs and feature types
```

Request flow:

```text
Request -> Chi route -> Feature controller -> Service -> Repository -> DB -> Service -> Controller -> Response
```

## Performance Features

- Server-side pagination: `/api/requests?page=1&per_page=20`
- API response cache: short TTL in-memory cache for request stats and request list pages
- Browser cache: static assets served with long-lived cache headers
- Background jobs: request notification writes run through `internal/jobs`
- Database indexes: login and request filter/search indexes are ensured on startup
- Rate limiting: `RATE_LIMIT_PER_MINUTE`

## Realtime

SSE endpoint:

```text
GET /api/events
```

WebSocket endpoint:

```text
GET /api/ws
```

Both publish workflow events from the in-process event bus.

## CI/CD

GitHub Actions workflow:

```text
.github/workflows/ci.yml
```

It runs:

- `go test ./...`
- `go build ./cmd/server`
- `go build ./cmd/frontend`
- `npm ci`
- `npm run build` in `frontend-react`

## Deployment Notes

- Supabase hosts Postgres.
- Vercel can host the React frontend.
- Cloudflare manages DNS.
- Namecheap can provide the domain.
- Better Stack can monitor `/healthz`.
- Sentry and PostHog are enabled by frontend env variables.
- Upstash Redis can replace the in-memory cache/rate limiter when the app runs multiple API replicas.
