---
name: Security and Performance Fixes
description: Key architectural decisions made during the security/performance audit and fix pass.
---

## Auth Design
JWT in HttpOnly SameSite=Strict cookie, signed with `APP_SECRET` env var (defaults to placeholder if unset — warn user to set it).
- Token issued in `LoginAPI`, cleared in `LogoutAPI` (POST /api/logout)
- Auth middleware in `internal/auth/jwt.go` via `RequireAuth()` applied to all `/api/*` routes except `/api/login` and `/api/logout`
- Session user available in handlers via `c.Get("auth_user").(*auth.SessionUser)`
- JWT expires in 12 hours (one shift)

**Why:** Original app had zero server-side auth — role/identity came entirely from localStorage.

## Frontend 401 Handling
`apiFetch()` wrapper in app.js intercepts 401, clears localStorage, and shows the login modal.
All non-login fetch calls use `apiFetch()`. Login itself uses plain `fetch()`.

**Why:** Without this, expired/missing sessions would silently fail instead of prompting re-login.

## Stats Query Collapse
`loadStats()` now runs a single `SELECT COUNT(*) FILTER (WHERE ...) FROM requests` query returning all 6 counters.
Page handlers call `loadStats()` once and reuse the result for `PendingOps/MM/Dock` badge counts.

**Why:** Original fired 6+ separate COUNT queries per page render → 10 DB roundtrips per page load.

## Timezone Fix
All "today" calculations and trend windows use `time.LoadLocation("Asia/Manila")` stored in `manilaLocation` package var (initialized in `init()`). Timestamps formatted with `.In(manilaLocation)`.

**Why:** Server runs UTC; original used `time.Now().Truncate(24h)` which was UTC midnight, not PH midnight.

## hourlyRequestTrend
Replaced in-memory row-fetch-and-count with a single `GROUP BY date_trunc('hour', ...)` SQL query.

## TOCTOU Fix
Removed `userIdentifierExists()` pre-check. `models.User` now has `uniqueIndex` on `email` and `ops_id`. `CreateUserAPI` handles the unique constraint violation from `Create()` via `isUniqueConstraintError()`.

## Role Authorization
`CreateUserAPI` reads actor role from the JWT claims (`c.Get("auth_user")`), not from the request body `actor_role` field. Client still sends it but server ignores it.

## Rate Limiter
Custom IP-based limiter in `internal/middleware/ratelimit.go` — 10 attempts per minute per IP on login. No external dependency needed.

## SSE Improvements
- Buffer increased from 16 to 32
- `X-Accel-Buffering: no` header added (prevents nginx from buffering the stream)
- `id:` field on every SSE event
- `Last-Event-ID` header support: `Bus.EventsSince()` replays up to 100 buffered events on reconnect

## Graceful Shutdown
`cmd/server/main.go` now runs Echo in a goroutine and waits for SIGTERM/SIGINT, then calls `e.Shutdown(ctx)` with 10s timeout.

## Deployment Config
`APP_SECRET` env var must be set in production for JWT security. Set it via Replit Secrets.
