# Backend Guide

## Overview

The backend is a Go/Echo API application. It serves JSON API endpoints, QR images, and server-sent events. The browser UI is a separate static frontend under `frontend/`.

Data is stored in PostgreSQL through GORM. Supabase Postgres is the expected hosted database, but any compatible Postgres database can work.

## Current Structure

```text
cmd/server/main.go        API server boot, env loading, CORS, migrations
cmd/frontend/main.go      local static frontend server
frontend/                 static browser app
internal/database/        Postgres/GORM connection setup
internal/routes/          API route registration
internal/handlers/        API handlers, auth, request workflow, user management
internal/events/          in-process pub/sub bus for request workflow SSE
internal/models/          GORM models for clusters, users, requests, events, notifications, OTPs
web/                       legacy frontend source kept for reference
docs/database.txt         database table reference
```

There is currently no separate `services` or `middleware` package. Most backend behavior is implemented in `internal/handlers/dashboard.go`, with SSE helpers in `internal/handlers/events.go`.

## Environment

| Variable | Required | Purpose |
|---|---:|---|
| `DATABASE_URL` | No | Preferred Postgres connection string. |
| `DB_DSN` | No | Alternative Postgres DSN string. |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | No | Alternative separate database connection fields. |
| `DB_SSLMODE` | No | Postgres SSL mode, defaults to `require`. |
| `JWT_SECRET` | No | HMAC secret for session JWT signing. Falls back to a development default if missing. |
| `FRONTEND_URL` | No | Allowed frontend CORS origin, defaults to `http://localhost:5173`. |
| `PORT` | No | Deployment port, checked before `APP_PORT`. |
| `APP_PORT` | No | Local port fallback, defaults to `8080`. |
| `APP_HOST` | No | Bind host, defaults to `127.0.0.1`. |
| `APP_ENV` | No | Set to `production` to mark auth cookies as secure. |

If database variables are missing, the app starts with `database.DB == nil` and several read paths return empty preview data. Mutating database APIs return `503 Database is not configured`.

## Startup Flow

1. `cmd/server/main.go` loads `.env` with `godotenv`.
2. `internal/database.Connect` opens a GORM Postgres connection when database settings are available.
3. When connected, startup runs `AutoMigrate` for:
   - `clusters`
   - `users`
   - `requests`
   - `request_events`
   - `notifications`
   - `user_otps`
4. `ensureWorkflowConstraints` normalizes legacy request statuses and recreates request status and user role check constraints.
5. Echo is configured with CORS and registered API routes.
6. The server binds to `APP_HOST:PORT` or `APP_HOST:APP_PORT`.

## Auth

Session auth is implemented with an HTTP-only cookie named `soc5_token`.

Supported endpoints:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/login` | Alias for login. |
| `POST` | `/api/auth/login` | Password login for FTE or Backroom users. |
| `POST` | `/api/auth/send-otp` | Creates a 6-digit OTP for active FTE users. |
| `POST` | `/api/auth/verify-otp` | Validates OTP and sets the session cookie. |
| `POST` | `/api/auth/logout` | Clears the session cookie. |
| `GET` | `/api/auth/me` | Returns session claims from the cookie. |
| `POST` | `/api/auth/change-password` | Updates the signed-in user's password. |

Login rules:

- FTE login uses email and supports `fte_ops` and `fte_mm`.
- Backroom login uses Ops ID and supports `ops_pic`, `dock_officer`, and `doc_officer`.
- If a user has `password_hash`, bcrypt verification is required.
- OTP is currently for FTE roles only.

Cookie properties:

- Name: `soc5_token`
- `HttpOnly`
- `SameSite=Lax`
- `Path=/`
- `Secure` only when `APP_ENV=production`
- JWT expiry is 12 hours

Important gap: API routes are not protected by centralized Echo middleware. Handlers that need identity call `readSessionClaims` directly. Many workflow APIs currently rely on frontend role gating and payload rules rather than server-side role middleware.

## Frontend Boundary

The backend no longer serves page routes or static assets. The standalone frontend files live in `frontend/` and can be served locally with:

```powershell
go run .\cmd\frontend
```

The frontend uses `frontend/config.js` to choose its API origin. Local separate development defaults to `http://localhost:8080`.

## Core API Routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | Backend health check. |
| `GET` | `/api/stats` | Dashboard counters. |
| `GET` | `/api/request-trend` | Hourly request trend data. |
| `GET` | `/api/requests` | List/filter requests. |
| `POST` | `/api/requests` | Create request. |
| `GET` | `/api/requests/:id` | Fetch one request. |
| `PUT` | `/api/requests/:id` | Edit request. |
| `POST` | `/api/requests/:id/edit` | Edit request alias. |
| `POST` | `/api/requests/:id/cancel` | Cancel request. |
| `POST` | `/api/requests/:id/approve` | Approve request. |
| `POST` | `/api/requests/bulk-approve` | Approve multiple pending/rejected requests. |
| `POST` | `/api/requests/:id/reject` | Reject request. |
| `POST` | `/api/requests/:id/reject-mm` | Reject request alias. |
| `POST` | `/api/requests/:id/assign` | Assign truck information. |
| `POST` | `/api/requests/:id/assign-truck` | Move truck to docking flow. |
| `POST` | `/api/requests/:id/for-docking` | Move truck to docking flow alias. |
| `POST` | `/api/requests/:id/dock` | Mark docked. |
| `POST` | `/api/requests/:id/mark-docked` | Mark docked alias. |
| `POST` | `/api/requests/:id/confirm` | Confirm request. |
| `GET` | `/api/requests/:id/events` | Request event history. |
| `GET` | `/api/clusters` | Cluster lookup options. |
| `GET` | `/api/qr` | Driver QR PNG. |

## Workflow Statuses

Current canonical statuses:

```text
PENDING
APPROVED
ASSIGNED
FOR_DOCKING
DOCKED
CONFIRMED
REJECTED_BY_MM
CANCELLED
```

Legacy values are normalized at startup:

| Legacy | Current |
|---|---|
| `PENDING_OPS` | `PENDING` |
| `PENDING_MM` | `APPROVED` |
| `REJECTED` | `REJECTED_BY_MM` |
| `CANCELED` | `CANCELLED` |

Action validation currently checks required fields:

| Action | Required data |
|---|---|
| reject/cancel | remarks |
| for-docking | plate number |
| dock | driver ID and LH trip number |

## Events, Notifications, And Realtime

Mutating request actions call `publishRequestEvent`.

When the database is configured, this writes a `request_events` row and creates role-targeted `notifications` for selected event types. It also publishes to the in-process event bus.

SSE endpoints:

| Path | Behavior |
|---|---|
| `/api/events` | Streams workflow events from the in-process event bus. |
| `/api/realtime/notifications` | Polls unread sound notifications every 3 seconds and emits `notification` events. |

Notification APIs:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/notifications` | List notifications, optionally filtered by role. |
| `PATCH` | `/api/notifications/:id/read` | Mark notification as read. |
| `PATCH` | `/api/notifications/:id/sound-played` | Clear the sound alert flag. |

Important limitation: `internal/events.DefaultBus` is in memory. It is best-effort on serverless or multi-instance hosting. Database-backed notifications are the more reliable path.

## User Management

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/users` | List users. |
| `POST` | `/api/users` | Create a user. |
| `PUT` | `/api/users/:id` | Update a user. |
| `PATCH` | `/api/users/:id/disable` | Disable a user. |

Current allowed roles in startup constraints:

```text
fte_ops
fte_mm
ops_pic
dock_officer
doc_officer
```

## Error Handling

The custom Echo error handler returns JSON in this shape:

```json
{ "error": "message" }
```

Common statuses:

- `400` for invalid payloads or failed business validation.
- `401` for missing/invalid login, session, or OTP.
- `403` for user-management role restrictions.
- `404` when a requested row is missing.
- `409` for duplicate user identifiers.
- `503` when database-dependent APIs are called without a configured database.
- `500` for database or server failures.

## Known Gaps

- Server-side role authorization is not centralized. Sensitive workflow and user-management routes should enforce session role checks in handlers or middleware.
- OTP generation stores the code hash but does not send the code through email yet.
- `JWT_SECRET` should be required in production instead of silently falling back to a development secret.
