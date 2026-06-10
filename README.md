# Golang Dashboard

SOC 5 outbound linehaul request dashboard with a separated Go API backend and static browser frontend.

## Requirements

- Go 1.25 or newer
- Postgres database, such as Supabase Postgres

## Project Layout

```text
cmd/server/              Backend API/SSE server
cmd/frontend/            Local static frontend server
frontend/                Static HTML/CSS/JS frontend app
internal/database/       Postgres connection setup
internal/events/         In-process event bus for workflow events
internal/handlers/       JSON API handlers
internal/models/         GORM models
internal/routes/         API route registration
web/                     Legacy frontend source kept for reference
docs/database.txt        Database table reference
```

## Setup

Install dependencies:

```powershell
go mod download
```

Create a `.env` file in the project root:

```env
APP_PORT=8080
APP_HOST=127.0.0.1
FRONTEND_URL=http://localhost:5173

DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
```

`DB_DSN` and separate `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` fields are also supported. If database variables are missing, the backend starts in preview mode with empty read data and database writes return `503`.

## Run Locally

Start the backend API:

```powershell
go run .\cmd\server
```

In another terminal, start the frontend:

```powershell
go run .\cmd\frontend
```

Open:

```text
http://localhost:5173/dashboard.html
```

The frontend reads `frontend/config.js`. For local separated development, it points API calls to `http://localhost:8080`.

## Main Frontend Pages

- `frontend/dashboard.html`
- `frontend/lh-request.html`
- `frontend/truck-request.html`
- `frontend/dock-officer.html`
- `frontend/settings.html`

## Main Backend Routes

- `GET /healthz`
- `POST /api/login`
- `GET /api/stats`
- `GET /api/requests`
- `POST /api/requests`
- `GET /api/events`
- `GET /api/realtime/notifications`

Login validates FTE users by email and Backroom users by Ops ID against the `users` table. Request workflow updates use `/api/events` with polling fallbacks in the frontend.

## Test

Run all tests:

```powershell
go test ./...
```

## Deployment

For separated frontend/backend deployment notes, see:

```text
docs/deployment.md
```
