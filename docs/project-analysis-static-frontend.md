# Project Analysis and Recommendations

Date: 2026-06-11

Status: Superseded by the 2026-06-11 switch back to the React/Vite frontend. Keep this document as historical analysis of the static-frontend option.

## Executive Summary

The project is now best treated as a Go API plus static frontend application. That direction fits the current codebase: the active UI lives in `frontend/`, the local frontend server is `cmd/frontend`, and Docker/NGINX are optional rather than required for development.

The backend has a reasonable feature-package structure and the main request workflow is already separated into controller, service, repository, model, cache, event, and job concerns. The main risks are not the static frontend choice. The larger gaps are server-side authorization, lack of automated tests, startup database migrations in production, frontend file size/duplication, and repository hygiene around generated artifacts.

## Current Structure

```text
cmd/server/              Go API entrypoint, middleware, DB startup, startup constraints
cmd/frontend/            Local static frontend server for frontend/
frontend/                Active static HTML/CSS/JS app
frontend-react/          Archived React/Vite prototype, no longer active path
internal/database/       GORM Postgres connection
internal/events/         In-process event bus
internal/jobs/           Background job queue
internal/features/       Feature packages for auth, requests, users, clusters, notifications, realtime, QR
internal/models/         GORM models
internal/routes/         API route registration
deploy/nginx.conf        Optional Docker/NGINX reverse proxy config
docs/                    Product, setup, deployment, backend, and system docs
web/                     Legacy template/static frontend source kept for reference
```

## Strengths

- The active local path is simple: `go run .\cmd\server` plus `go run .\cmd\frontend`.
- The backend is organized by business feature, which is easier to maintain than grouping by technical layer only.
- Request lifecycle behavior is centralized mostly in `internal/features/requests/service.go`.
- Database access is isolated behind repositories, which makes future testing and refactoring easier.
- The static frontend avoids a Node/Vite build requirement for day-to-day development.
- The app already has useful production concerns in place: CORS, security headers, rate limiting, cache headers, background jobs, SSE/WebSocket support, Docker targets, and deployment docs.

## High Priority Gaps

### 1. Server-side authorization is not centralized

Most workflow routes are registered directly in `internal/routes/routes.go` without visible auth or role middleware. The frontend hides actions by role, but frontend role checks are only UX controls, not security controls.

The biggest concrete issue is user management. `internal/features/users/service.go` authorizes create/update/disable through `payload.ActorRole`, which comes from the browser. A client can forge this field unless the server derives actor role from the signed session cookie.

Recommendation:

- Add auth middleware that reads `soc5_token`, validates the signature/expiry, and stores claims in request context.
- Add role middleware for workflow actions:
  - `fte_ops` / `ops_pic`: create, edit pending, approve, cancel where appropriate.
  - `fte_mm`: reject, assign, move to docking.
  - `dock_officer` / `doc_officer`: dock and confirm.
  - `fte_ops` / `fte_mm`: user management.
- Remove `actor_role` from trusted request payloads. Derive it only from server-side claims.

### 2. There are no Go tests

`rg --files -g "*_test.go"` found no test files. `go test ./...` passes because there are no test cases.

Recommendation:

- Start with service-level tests for the request status machine:
  - create request
  - edit allowed only for pending/rejected
  - approve
  - reject
  - assign
  - for docking
  - dock
  - confirm
  - invalid transition behavior
- Add auth tests for password login, OTP verification, session expiry, and invalid JWT signatures.
- Add users tests specifically for server-side role enforcement after fixing `actor_role`.

### 3. Production startup migrations are risky

`cmd/server/main.go` runs `AutoMigrate`, drops/recreates constraints, and creates indexes during startup. This is convenient locally, but risky in production because every app start can modify schema.

Recommendation:

- Keep `AutoMigrate` only for local development.
- Add a migration command or migration folder for controlled database changes.
- Gate startup migrations behind an explicit env var such as `AUTO_MIGRATE=true`.
- Document the production migration process separately.

### 4. JWT secret has a development fallback

`internal/features/auth/service.go` falls back to `soc5-dev-session-secret` if `JWT_SECRET` and `APP_SECRET` are empty. That is useful locally but unsafe if a production environment is misconfigured.

Recommendation:

- In `APP_ENV=production`, fail startup if `JWT_SECRET` is missing or too short.
- Keep the development fallback only for non-production.
- Add this requirement to `.env.example`, README, setup guide, and deployment docs.

### 5. Static frontend is large and duplicated

The active frontend is static, but `frontend/static/app.js` is about 1,635 lines and `frontend/static/app.css` is about 2,528 lines. HTML pages also repeat common shell/sidebar/topbar/modal markup.

Recommendation:

- Keep static frontend, but split `app.js` by concern:
  - `api.js`
  - `auth.js`
  - `requests.js`
  - `notifications.js`
  - `settings.js`
  - `ui.js`
- Split CSS into base/layout/components/pages.
- Consider a tiny static build helper later only if duplication becomes painful. This does not require moving to React.

## Medium Priority Gaps

### 6. Two HTTP frameworks are mixed

The app routes with Chi, but feature controllers use Echo handlers through an adapter in `internal/routes/routes.go`. This works, but it increases cognitive load and creates one Echo instance per adapted handler.

Recommendation:

- Standardize on Chi-native handlers over time.
- Convert controllers gradually when touching a feature.
- Keep the existing adapter until migration is complete.

### 7. In-memory cache and event bus are single-instance only

The request cache, rate limiter, job queue, and event bus are in-process. That is fine for one local or single-instance deployment, but not for multiple replicas.

Recommendation:

- Document single-instance assumptions clearly.
- Use Supabase/Postgres polling, Redis/Upstash, or another shared system before scaling horizontally.
- Keep current in-memory implementation for the free/local stack.

### 8. Docker Compose still exists but is optional

This is fine, but the chosen path is static local development without Docker. Docker should remain a deployment/testing option, not a requirement.

Recommendation:

- Keep Docker files.
- Ensure docs consistently say Docker is optional.
- Do not make Docker the primary onboarding path for this project.

### 9. Repository hygiene needs cleanup

The working tree contains generated or local artifacts:

```text
server.exe
frontend.exe
server.out.log
server.err.log
ui-server.out.log
ui-server.err.log
node_modules/
package-lock.json
frontend-react/node_modules/
frontend-react/dist/
frontend-react/*.tsbuildinfo
```

`.gitignore` already ignores `.env`, `*.exe`, `frontend-react/node_modules/`, `frontend-react/dist/`, and `frontend-react/*.tsbuildinfo`, but it does not ignore root `node_modules/`, root logs, or root `package-lock.json`.

Recommendation:

- Add these ignores:

```gitignore
node_modules/
*.log
package-lock.json
```

- Remove local generated artifacts from the workspace when safe.
- Keep `frontend-react/` only if it is intentionally archived reference material; otherwise delete it in a separate cleanup commit.

## Low Priority Improvements

### 10. Static frontend server fallback behavior is minimal

`cmd/frontend` maps `/` to `/index.html`, but it does not rewrite clean URLs like `/dashboard` to `dashboard.html`. The frontend JS has a route map, but direct browser navigation to clean paths may still depend on client-side links.

Recommendation:

- Either document `.html` URLs as canonical, or add server-side fallback mapping for known routes.

### 11. Observability is mostly logs

The app logs startup and some background failures, but request metrics, structured logs, and trace IDs are limited.

Recommendation:

- Keep simple logs for now.
- Add structured request logging before production.
- Expose basic app metrics only if there is an actual operations need.

### 12. Documentation is broad but partially overlapping

There are many docs: setup, backend, deployment, AWS, Vercel, system design, blueprint, PRD, functions summary. Some overlap and can drift.

Recommendation:

- Treat `README.md` and `docs/setup-guide.md` as the canonical local setup docs.
- Treat `docs/backend.md` as canonical API/backend behavior.
- Move old or exploratory docs under an `archive` section if they are no longer active.

## Recommended Implementation Plan

### Phase 1: Security and cleanup

1. Add centralized session parsing middleware.
2. Add role authorization middleware.
3. Remove trust in client-provided `actor_role`.
4. Require `JWT_SECRET` in production.
5. Update `.gitignore` for root `node_modules/`, logs, and root `package-lock.json`.

### Phase 2: Tests

1. Add request service tests for workflow transitions.
2. Add auth/session tests.
3. Add users service tests for role-gated actions.
4. Add a basic HTTP route test for `GET /healthz` and one protected route.

### Phase 3: Static frontend maintainability

1. Split `frontend/static/app.js` by feature.
2. Split `frontend/static/app.css` by concern.
3. Extract repeated HTML shell pieces using a simple generation script only if duplication becomes expensive.

### Phase 4: Production readiness

1. Move schema changes out of automatic startup execution.
2. Add explicit migration process.
3. Document single-instance limits for cache, rate limiter, jobs, and realtime.
4. Add structured logs and deployment smoke checks.

## Static Frontend Decision

Staying static is a good fit right now. The project does not need a full React migration to be useful or maintainable. The better near-term investment is to harden backend authorization, add tests, and modularize the existing static frontend.

React should stay archived unless one of these becomes true:

- The app needs many complex reusable UI components.
- Frontend state becomes too hard to manage in plain JavaScript.
- The team needs TypeScript-level frontend safety.
- Complex client-side routing becomes a real requirement.

Until then, the active stack should remain:

```text
Go API + Supabase Postgres + static frontend + optional Docker/NGINX
```
