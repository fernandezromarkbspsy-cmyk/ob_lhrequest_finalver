---
name: Full Stack Migration
description: Architecture decisions and constraints from the React+TS rewrite with Redis, Docker, NGINX, Cloudflare.
---

## Dev Architecture
- **Frontend**: Vite dev server on port 5000 (`web/frontend/`, `cd web/frontend && npm run dev`)
- **Backend**: Go API server on port 8080 (`go run ./cmd/server`)
- Vite proxies `/api/*` and `/truck_label/*` to `http://localhost:8080`
- Two Replit workflows: "Start application" (webview, port 5000) + "Backend" (console, port 8080)

## React App Structure
- `web/frontend/src/` — TypeScript React SPA
- Routing: react-router-dom v6, BrowserRouter with `v7_startTransition` and `v7_relativeSplatPath` future flags enabled
- Server state: TanStack Query v5 (`@tanstack/react-query`)
- Client state: Zustand v5 with `persist` middleware (localStorage keys: `soc5_user`, `soc5_ui`)
- Styling: SASS with `@use` module system — each partial must include `@use 'variables' as *;` at the top
- SSE: `useSSE` hook in `src/hooks/useSSE.ts` — subscribes to `/api/events`, invalidates queries on events
- Icons: `web/truck_label/icon.png` (delivery truck, copied from web/static/icons/delivery.png)

**Why:** Old frontend was Go HTML templates + 1400-line vanilla JS. Replaced with typed, component-based SPA.

## Redis Caching (Go backend)
- Package: `github.com/redis/go-redis/v9` in `internal/cache/redis.go`
- Stats cached 15s (`soc5:stats`), clusters cached 5min (`soc5:clusters`)
- Stats cache invalidated on every request mutation (in `publishRequestEvent`)
- Redis unavailable → graceful fallback to direct DB queries (no crash)
- Multi-instance SSE: mutations publish to Redis `soc5:events` channel; each instance's subscriber goroutine distributes to local bus → SSE clients

**Why:** Stats was queried on every API call. Redis removes DB roundtrip for the 15s polling window.

## Production Architecture
- `docker/Dockerfile` — multi-stage: Go build → alpine runtime
- `docker/docker-compose.yml` — redis + api + nginx services
- `docker/nginx.conf` — serves React SPA + proxies /api to Go, SSE buffering disabled
- Go serves `web/dist/` static files if `web/dist/index.html` exists (auto-detected at startup)
- Build React: `cd web/frontend && npm run build` → outputs to `web/dist/`

## Cloudflare Setup
- Full setup guide at `docs/cloudflare-setup.md`
- SSL mode: Full (strict) — requires valid cert on server side
- Cache rule: `/assets/*` = 1yr (Vite hashes filenames), `/api/*` = bypass

## SASS Module System Constraint
All SCSS partials that use SASS variables (`$font-stack`, `$sidebar-bg`, `$transition-*`, etc.) MUST start with:
```scss
@use 'variables' as *;
```
Without this, Vite's sass compiler throws "Undefined variable" errors. The `@use` rule does NOT automatically make parent imports' variables available in child files.
