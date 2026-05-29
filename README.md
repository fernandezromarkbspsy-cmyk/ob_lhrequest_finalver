# Golang Dashboard

SOC 5 outbound linehaul request dashboard built with Go, Echo, GORM, Postgres, server-rendered HTML templates, and vanilla browser JavaScript.

## Requirements

- Go 1.25 or newer
- Postgres database, such as Supabase Postgres

## Setup

1. Install dependencies:

```powershell
go mod download
```

2. Create a `.env` file in the project root.

Use a single connection string:

```env
APP_PORT=8080
APP_HOST=127.0.0.1

DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
```

Or use separate connection fields:

```env
APP_PORT=8080
APP_HOST=127.0.0.1

DB_HOST=your-postgres-host
DB_PORT=5432
DB_NAME=your-database-name
DB_USER=your-database-user
DB_PASSWORD=your-database-password
DB_SSLMODE=require
```

`DB_DSN` is also supported if you prefer a Postgres DSN string. If database variables are missing, the app starts in preview mode with empty datasets.

3. Review the expected database structure:

```powershell
Get-Content docs\database.txt
```

The app maps the documented `Cluster`, `User`, and `Request` tables through GORM models and runs `AutoMigrate` on startup.

## Run

Start the server:

```powershell
go run .\cmd\server
```

Open the dashboard:

```text
http://localhost:8080/dashboard
```

If `APP_PORT` is changed in `.env`, use that port instead. `APP_HOST` defaults to `127.0.0.1` so local development does not require Windows firewall/admin changes. Use `APP_HOST=0.0.0.0` only when the app must be reachable from other machines and the firewall/network allows inbound traffic.

## Main Routes

- `/` dashboard
- `/dashboard` role-aware request stats and recent activity
- `/outbound/lh-request` Ops PIC creation and FTE Ops approval queue
- `/midmile/truck-request` FTE MM confirmation queue

Login validates FTE users by email and Backroom users by Ops ID against the `User` table. The dashboard also subscribes to `/api/events` through server-sent events for request workflow updates, while polling remains as a fallback.

## Test

Run all tests:

```powershell
go test ./...
```

## Project Structure

```text
cmd/server/              Application entry point
internal/database/       Postgres connection setup
internal/events/         In-process event bus for workflow events
internal/handlers/       HTTP handlers
internal/models/         GORM models
internal/routes/         Echo route registration
web/templates/           HTML templates
docs/database.txt        Database table reference
```
