# Free-Stack System Design

## Architecture

The project is a feature-organized modular monolith with event-driven workflow updates. The backend uses Chi for routing, GORM for Supabase Postgres access, bcrypt for password hashing, signed HTTP-only cookies for local auth, in-process caching, a background job queue, Server-Sent Events, and WebSockets.

```text
Browser -> NGINX -> Chi API -> Service/handler logic -> Repository queries -> Supabase Postgres
                     |
                     +-> Event bus -> SSE clients
```

The current frontend is the React + TypeScript app in `frontend-react`. Local development uses Vite on port `5173`; production preview and Docker builds serve `frontend-react/dist` through `cmd/frontend`.

## Request Flow

```text
Request -> Chi router -> feature controller -> service -> repository -> Supabase Postgres -> service -> controller -> response
```

Active API code is grouped by feature:

```text
internal/features/auth/
internal/features/requests/
internal/features/users/
internal/features/clusters/
internal/features/notifications/
internal/features/realtime/
internal/features/qr/
```

Controllers handle HTTP, services handle business rules, and repositories handle data access.

## Data and Scale

- Supabase Postgres is the primary database.
- API list queries use server-side pagination and limits to avoid returning unbounded rows.
- Short-lived in-memory API caching reduces repeated stats and request-list queries.
- Static assets use long-lived browser cache headers.
- Background jobs move notification writes out of the user-facing request path.
- API requests are rate-limited per client IP using `RATE_LIMIT_PER_MINUTE`.
- NGINX can sit in front of multiple API replicas later and distribute traffic with round robin or least connections.

## Reliability

- `/healthz` supports container, NGINX, and external uptime checks.
- Docker targets package API and frontend separately.
- Database credentials, secrets, API keys, and ports are configured through environment variables.
- Supabase automated backups should be enabled from the Supabase project settings.

## Security

- Passwords and OTPs are stored with bcrypt hashes.
- Session data is signed and stored in HTTP-only cookies.
- CORS is restricted to `FRONTEND_URL`.
- Security headers are applied by both the Go API and NGINX.
- Client input is bound into typed payloads and validated before write operations.
- WebSocket origin checks use `FRONTEND_URL`.

## Optional Free-Tier Integrations

- Clerk can replace local auth when `CLERK_SECRET_KEY` is configured.
- Resend can deliver OTP and workflow emails when `RESEND_API_KEY` is configured.
- PostHog can collect product analytics from the frontend.
- Sentry can collect frontend and backend errors.
- Upstash Redis can replace the in-memory rate limiter/cache for multi-replica deployments.
- Pinecone is reserved for future document/search workflows, not core truck requests.
- Better Stack can monitor `/healthz` and container logs.
