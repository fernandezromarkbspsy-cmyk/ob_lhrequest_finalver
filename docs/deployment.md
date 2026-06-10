# Deployment Setup Guide

This guide covers deploying the SOC 5 outbound dashboard as a separated frontend/backend app, with Supabase Postgres as the database.

The project now has two runtime surfaces:

- Backend: Go/Echo API and SSE server from `cmd/server`.
- Frontend: static HTML/CSS/JS app from `frontend/`.

## Deployment Target

Recommended setup:

- Frontend hosting: Vercel Hobby static project, Netlify, Cloudflare Pages, or any static host
- Backend hosting: a Go-capable host such as Vercel Go runtime, Render, Fly.io, Railway, or another container/server host
- Database: Supabase Postgres
- Deployment type while in development: Preview deployment first, then Production when the workflow is verified

Important Vercel behavior:

- Vercel Hobby is suitable for personal or development projects.
- Hobby deployments can be paused if free tier usage is exceeded.
- If the backend is deployed on Vercel, Go server support requires the Go framework preset.
- The backend must listen on the host-provided `PORT` environment variable.
- Long-lived realtime connections such as SSE are limited by function duration and serverless lifecycle behavior.

Official references:

- Vercel Go runtime: https://vercel.com/docs/functions/runtimes/go
- Vercel function limits: https://vercel.com/docs/functions/limitations
- Vercel plans: https://vercel.com/docs/plans
- Vercel environment variables: https://vercel.com/docs/environment-variables

## Readiness Checklist

Before deploying, confirm these items:

- `go.mod` and `go.sum` are committed.
- `cmd/server/main.go` and `cmd/frontend/main.go` are committed.
- `frontend`, including `frontend/static` and `frontend/truck_label`, is committed.
- `.env` is not committed.
- Supabase database is created.
- Required tables can be created or migrated by the app.
- At least one login user exists in the `users` table.
- The backend listens on `PORT` in deployed environments.
- `FRONTEND_URL` is set on the backend to the deployed frontend origin.
- `frontend/config.js` points `API_BASE` to the deployed backend origin.
- Production secrets are stored in Vercel Project Settings, not in source code.

## Required Code Compatibility

Vercel sets the runtime port through `PORT`. For Vercel deployment, the server must prefer `PORT` over local `APP_PORT`.

The server startup should follow this behavior:

```go
port := os.Getenv("PORT")
if port == "" {
    port = os.Getenv("APP_PORT")
}
if port == "" {
    port = "8080"
}

host := os.Getenv("APP_HOST")
if host == "" && os.Getenv("PORT") == "" {
    host = "127.0.0.1"
}

addr := ":" + port
if host != "" {
    addr = host + ":" + port
}
```

This keeps local backend development on `127.0.0.1:8080`, while allowing hosted deployments to bind correctly.

## Frontend Configuration

The frontend API origin is configured in:

```text
frontend/config.js
```

For local development, it automatically uses:

```text
http://localhost:8080
```

For deployment, set `API_BASE` to the backend URL:

```js
window.SOC5_CONFIG = {
  API_BASE: "https://your-backend.example.com"
};
```

On the backend host, set:

```env
FRONTEND_URL=https://your-frontend.example.com
```

## Supabase Database Setup

1. Create a Supabase project.
2. Open the Supabase dashboard.
3. Go to Project Settings.
4. Open Database.
5. Copy the pooled or direct Postgres connection string.
6. Make sure SSL is required.

Use a connection string like:

```env
DATABASE_URL=postgres://postgres.your-project-ref:your-password@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require
```

Notes:

- Use the pooler connection string for serverless hosting when possible.
- Keep `sslmode=require`.
- Do not expose Supabase service role keys in frontend JavaScript.
- This project connects directly from the Go backend, so the frontend does not need Supabase keys.

## Database Schema

Review the expected schema:

```powershell
Get-Content docs\database.txt
```

The app uses GORM models and currently runs `AutoMigrate` during startup when the database connection succeeds.

For development deployments, this is convenient.

For production or shared operational data, consider replacing automatic startup migrations with a controlled migration process before exposing the app to real users.

Minimum tables expected by the current app:

- `clusters`
- `users`
- `requests`
- `request_events`
- `notifications`

## Seed First Login User

Before testing login, insert at least one active user into Supabase.

The app supports these role values:

- `ops_pic`
- `fte_ops`
- `fte_mm`
- `dock_officer`
- `doc_officer`

Typical test users:

- FTE Ops user for approving and managing requests.
- FTE MM user for assigning trucks.
- Ops PIC user for creating requests.
- Dock Officer user for docking and confirmation.

Use the schema in `docs/database.txt` and the model fields in `internal/models/user.go` as the source of truth for required columns.

## Local Verification Before Deploy

Install dependencies:

```powershell
go mod download
```

Create `.env` locally:

```env
APP_PORT=8080
APP_HOST=127.0.0.1
DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
```

Run tests:

```powershell
go test ./...
```

Run the backend:

```powershell
go run .\cmd\server
```

Run the frontend in another terminal:

```powershell
go run .\cmd\frontend
```

Open:

```text
http://localhost:5173/dashboard.html
```

Verify:

- Frontend dashboard loads.
- Static CSS and JS load.
- Login works.
- `/api/auth/me` returns the current user after login.
- `/api/requests` returns data or an empty list without crashing.
- Creating, approving, assigning, docking, and confirming requests work.

## Vercel Project Setup

1. Push the repository to GitHub.
2. Open Vercel.
3. Select Add New Project.
4. Import the GitHub repository.
5. For backend deployment, select the Go framework preset and use `cmd/server`.
6. For frontend deployment, deploy the `frontend` directory as static files.
7. Confirm that `go.mod` is at the repository root for the backend.
8. Deploy first as a Preview deployment.

Vercel should detect this entry point:

```text
cmd/server/main.go
```

If Vercel does not detect Go correctly, set the Framework Preset manually to `Go`.

## Vercel Environment Variables

Add these in Vercel Project Settings, under Environment Variables.

For Preview and Production:

```env
DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
APP_ENV=production
FRONTEND_URL=https://your-frontend.example.com
```

Usually do not set these on Vercel:

```env
APP_HOST=127.0.0.1
APP_PORT=8080
```

Vercel provides `PORT` automatically. Setting `APP_HOST=127.0.0.1` in Vercel can break public access if the server binds only to localhost.

If a future auth implementation requires JWT signing, also add:

```env
JWT_SECRET=generate-a-long-random-secret
```

After changing environment variables, redeploy. Vercel environment changes only apply to new deployments.

## Suggested Vercel Settings

Recommended for this project:

- Framework Preset: `Go`
- Root Directory: repository root
- Build Command: leave default unless Vercel requires one
- Output Directory: leave empty
- Install Command: leave default
- Production Branch: `main`

Optional `vercel.json`:

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "build": {
    "env": {
      "GO_BUILD_FLAGS": "-ldflags '-s -w'"
    }
  }
}
```

Do not add this file unless you need explicit build flags. Vercel can build this project from `go.mod`.

## Preview Deployment Flow

Use this flow while the project is still in development:

1. Create a branch, for example `deploy-preview`.
2. Push the branch to GitHub.
3. Let Vercel create a Preview deployment.
4. Use a separate Supabase database or schema for preview data.
5. Test all user roles and request statuses.
6. Fix issues in the branch.
7. Merge to `main` only after the preview deployment is stable.

Recommended Preview environment variables:

```env
DATABASE_URL=preview-database-url
APP_ENV=production
```

Recommended Production environment variables:

```env
DATABASE_URL=production-database-url
APP_ENV=production
```

Using separate databases prevents development tests from modifying production data.

## Production Deployment Flow

When ready:

1. Confirm tests pass locally.
2. Confirm Preview deployment works.
3. Confirm Supabase production database is backed up.
4. Confirm production users are seeded.
5. Merge to the Vercel production branch, usually `main`.
6. Confirm the Production deployment succeeds.
7. Open the production URL.
8. Login as each role.
9. Run one full request workflow.
10. Monitor Vercel logs and Supabase logs.

## Post-Deployment Verification

Check these frontend URLs on the frontend host:

```text
/dashboard.html
/lh-request.html
/truck-request.html
/dock-officer.html
/settings.html
```

Check these backend URLs on the backend host:

```text
/healthz
/api/auth/me
/api/stats
/api/requests
/api/events
```

Expected results:

- Frontend pages should render from the static host.
- Static files under `frontend/static` should load.
- Truck label images under `frontend/truck_label` should load.
- API routes should return JSON.
- Unauthorized routes should fail cleanly when not logged in.
- Login should set the auth cookie.
- Role-specific pages should show the correct data and actions.

## Vercel Hobby Limitations

Vercel Hobby can be used for development and light preview usage, but there are constraints:

- Free tier usage is limited.
- Hobby projects can be paused after exceeding included usage.
- Function request duration is limited.
- Long-lived SSE connections may disconnect.
- Runtime logs are limited.
- Serverless instances can restart at any time.
- In-memory state is not durable across instances.

For this project, the most important limitation is the in-process realtime/event behavior. The `internal/events` bus is useful locally, but Vercel can run multiple instances and can restart them. Events stored only in memory are not guaranteed to reach all connected clients.

The current polling fallback helps. For stronger production realtime, use one of these:

- Supabase Realtime
- Postgres-backed notification polling
- Redis pub/sub
- A dedicated websocket service

## Security Checklist

Before exposing the deployment outside a small test group:

- Do not commit `.env`.
- Rotate any secret that was ever committed or shared.
- Use HTTPS only.
- Use strong database passwords.
- Use least-privilege database credentials where possible.
- Keep Supabase service role keys out of frontend code.
- Restrict Supabase network access if your plan and architecture allow it.
- Confirm cookies are `HttpOnly`.
- Confirm production cookies use `Secure`.
- Disable or protect any debug endpoints before real production use.
- Seed only real users who should have access.
- Disable inactive users.

## Operational Notes

The app currently runs database migrations at startup through GORM `AutoMigrate`.

That is acceptable for development and preview deployments. For production, prefer an explicit migration step so schema changes are reviewed before deployment.

The app also has server-sent events under:

```text
/api/events
```

Treat this as best-effort realtime on Vercel Hobby. The frontend should continue to use polling fallback for reliable updates.

## Troubleshooting

### Build does not detect Go

Confirm:

- `go.mod` exists at the repository root.
- `cmd/server/main.go` exists.
- Vercel Framework Preset is set to `Go`.

### Backend deployment succeeds but API does not load

Check:

- The app listens on `PORT`.
- `APP_HOST` is not set to `127.0.0.1` in Vercel.
- Vercel runtime logs show `Server running`.

### Frontend loads but API calls fail

Check:

- `frontend/config.js` has the correct backend `API_BASE`.
- Backend `FRONTEND_URL` exactly matches the frontend origin.
- Browser devtools do not show CORS errors.
- Backend `/healthz` returns JSON.

### Database connection fails

Check:

- `DATABASE_URL` is set in Vercel for the correct environment.
- The connection string includes `sslmode=require`.
- Supabase database password is correct.
- The Supabase project is active.
- The connection string is not URL-escaped incorrectly.

### Login fails

Check:

- A user exists in the `users` table.
- The user is active.
- The login identifier matches the user type.
- FTE users use email.
- Backroom users use Ops ID.
- Password or credential logic matches the current handler implementation.

### Static assets are missing

Check:

- `frontend/static` is committed.
- `frontend/truck_label` is committed.
- The frontend host is serving files from the `frontend` directory.

### Realtime updates disconnect

This is expected sometimes on Vercel Hobby. Confirm the frontend polling fallback is active. For production-grade realtime, move notification delivery to Supabase Realtime or a shared backend service.

## Rollback

If a deployment breaks:

1. Open the Vercel project.
2. Go to Deployments.
3. Select the last known working deployment.
4. Promote it back to Production.
5. Check Supabase data for partial writes caused by the broken deployment.

If the issue involved schema changes, restore from a Supabase backup or run a corrective migration.

## Minimal Deployment Summary

For a first development deployment:

1. Make sure the backend listens on `PORT`.
2. Push the repo to GitHub.
3. Deploy the backend from `cmd/server`.
4. Deploy the frontend from `frontend`.
5. Add `DATABASE_URL`, `APP_ENV=production`, and `FRONTEND_URL` to the backend environment.
6. Set frontend `API_BASE` to the backend URL.
7. Deploy as Preview.
8. Test login and the full request lifecycle.
9. Promote or merge to Production only after Preview is stable.
