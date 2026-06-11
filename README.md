# Golang Dashboard

SOC 5 outbound linehaul truck request portal. The app replaces spreadsheet-based request tracking with a Go API, Postgres persistence, realtime workflow updates, and role-aware frontend pages for Ops, Midmile, and Dock teams.

## Current Scope

- Role-aware login for FTE email users and Backroom/Ops ID users.
- Request lifecycle: `PENDING -> APPROVED -> ASSIGNED/FOR_DOCKING -> DOCKED -> CONFIRMED`.
- Reject, cancel, edit, bulk approve, assign truck, dock, and confirm workflows.
- Dashboard stats, request trend data, notifications, and request event history.
- Server-sent events for realtime updates with frontend polling fallback.
- API rate limiting, security headers, CORS configuration, and static asset cache headers.
- Dockerized backend, static frontend, and optional local Nginx reverse proxy.
- Deployment guides for AWS EC2 backend and Vercel/static frontend hosting.

## Requirements

- Go 1.25 or newer
- Postgres database, such as Supabase Postgres
- Docker Desktop, optional for containerized local runs
- Node.js, optional only for the `frontend-react/` prototype

## Project Layout

```text
cmd/server/              Go API and SSE backend
cmd/frontend/            Local static frontend server
frontend/                Main Gentelella-inspired vanilla JS frontend
frontend-react/          Optional React/Vite frontend prototype
internal/database/       Postgres connection setup
internal/events/         In-process event bus for workflow updates
internal/features/       Feature packages with controller/service/repository layers
internal/models/         GORM models
internal/routes/         API route registration
deploy/nginx.conf        Local reverse proxy config for Docker Compose
docs/                    Database, PRD, deployment, and system design docs
web/                     Legacy frontend source kept for reference
```

## Setup

Install Go dependencies:

```powershell
go mod download
```

Create a local `.env` from the example:

```powershell
Copy-Item .env.example .env
```

Minimum local values:

```env
APP_ENV=development
APP_HOST=127.0.0.1
APP_PORT=8080
FRONTEND_URL=http://localhost:5173
JWT_SECRET=replace-with-a-long-random-secret
RATE_LIMIT_PER_MINUTE=120

DATABASE_URL=postgres://user:password@db.project-ref.supabase.co:5432/postgres?sslmode=require

FRONTEND_HOST=127.0.0.1
FRONTEND_PORT=5173
FRONTEND_DIR=frontend
```

`DB_DSN` and separate `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, and `DB_SSLMODE` values are also supported. If database variables are missing, the backend starts in preview mode with empty read data and database writes return `503`.

## Run Locally

Start the backend API:

```powershell
go run .\cmd\server
```

Start the static frontend in another terminal:

```powershell
go run .\cmd\frontend
```

Open:

```text
http://localhost:5173/dashboard.html
```

The frontend uses `frontend/config.js` to choose its backend API origin. For separated local development on port `5173`, it calls:

```text
http://localhost:8080
```

## Docker Compose

Run the backend, static frontend, and Nginx reverse proxy:

```powershell
docker compose up --build
```

Local services:

```text
Backend API:      http://localhost:8080
Static frontend:  http://localhost:5173
Nginx proxy:      http://localhost:8088
```

The Docker setup uses `Dockerfile` build targets:

- `backend`: builds and runs `cmd/server`
- `frontend`: builds and runs `cmd/frontend`

## Optional React Frontend

`frontend-react/` is a Vite/React prototype that uses React Query, Zustand, Clerk, Sentry, PostHog, and Sass dependencies. It is separate from the main static frontend.

```powershell
cd frontend-react
npm install
npm run dev
```

The React dev server defaults to:

```text
http://127.0.0.1:5174
```

## Main Static Frontend Pages

- `frontend/index.html`
- `frontend/dashboard.html`
- `frontend/lh-request.html`
- `frontend/truck-request.html`
- `frontend/dock-officer.html`
- `frontend/settings.html`

## Main Backend Routes

- `GET /healthz`
- `POST /api/login`
- `POST /api/auth/login`
- `POST /api/auth/send-otp`
- `POST /api/auth/verify-otp`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/auth/change-password`
- `GET /api/stats`
- `GET /api/request-trend`
- `GET /api/requests`
- `POST /api/requests`
- `GET /api/requests/{id}`
- `PUT /api/requests/{id}`
- `POST /api/requests/{id}/cancel`
- `POST /api/requests/{id}/approve`
- `POST /api/requests/bulk-approve`
- `POST /api/requests/{id}/reject`
- `POST /api/requests/{id}/assign`
- `POST /api/requests/{id}/for-docking`
- `POST /api/requests/{id}/dock`
- `POST /api/requests/{id}/confirm`
- `GET /api/requests/{id}/events`
- `GET /api/clusters`
- `GET /api/qr`
- `GET /api/notifications`
- `PATCH /api/notifications/{id}/read`
- `PATCH /api/notifications/{id}/sound-played`
- `GET /api/events`
- `GET /api/realtime/notifications`
- `GET /api/users`
- `POST /api/users`
- `PUT /api/users/{id}`
- `PATCH /api/users/{id}/disable`

## Runtime Behavior

- The backend prefers `PORT` in hosted environments, then falls back to `APP_PORT`, then `8080`.
- Local backend binding defaults to `127.0.0.1`; hosted deployments should normally leave `APP_HOST` unset or use the platform default.
- CORS allows the exact origin in `FRONTEND_URL`.
- API rate limiting defaults to `120` requests per minute per client and can be changed with `RATE_LIMIT_PER_MINUTE`.
- `/api/events` and `/api/realtime/notifications` are exempted from the normal request timeout for long-lived realtime connections.
- Static files under `/static/` and `/truck_label/` get long-lived cache headers from the local frontend server.

## Database

The backend uses GORM models and runs `AutoMigrate` at startup when the database connection succeeds. During startup it also normalizes legacy request statuses and refreshes status/role check constraints.

Reference docs:

```text
docs/database.txt
docs/prd.md
```

Supported user roles include:

- `ops_pic`
- `fte_ops`
- `fte_mm`
- `dock_officer`
- `doc_officer`

Seed at least one active user before testing login.

## Test

Run all Go tests:

```powershell
go test ./...
```

Build the optional React frontend:

```powershell
cd frontend-react
npm run build
```

## Deployment

Separated frontend/backend deployment:

```text
docs/deployment.md
```

AWS EC2 backend with Nginx, systemd, and HTTPS:

```text
docs/aws-deployment.md
```

Vercel static frontend deployment:

```text
docs/vercel-frontend.md
```

System design and free-tier stack notes:

```text
docs/system-design-free-stack.md
```

## Security Notes

- Do not commit `.env`.
- Keep `DATABASE_URL`, `JWT_SECRET`, and third-party API keys outside source control.
- Set `FRONTEND_URL` to the exact deployed frontend origin.
- Use HTTPS in deployed environments.
- Keep Supabase service role keys out of frontend JavaScript.
- For production data, replace automatic startup migrations with a controlled migration process.
- For production-grade realtime across multiple backend instances, use a shared event system such as Supabase Realtime, Postgres-backed polling, or Redis pub/sub.
