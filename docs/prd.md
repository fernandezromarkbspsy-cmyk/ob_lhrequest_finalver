# Truck Request Portal PRD

## Problem

SOC 5 outbound linehaul work is moving from Google Sheets to an internal web app so Ops, Midmile, and Dock teams can request trucks, approve work, assign plates, capture docking details, and monitor queue health in real time.

## Users

- Ops PIC: creates and tracks outbound linehaul requests.
- FTE Ops: approves, rejects, edits, and manages Ops-side users.
- FTE Midmile: assigns truck resources and manages Midmile-side users.
- Dock Officer: captures driver, trip, and docking details.

## Core Features

- Role-aware login for FTE email users and Backroom Ops ID users.
- Truck request workflow: `PENDING -> APPROVED -> ASSIGNED/FOR_DOCKING -> DOCKED -> CONFIRMED`.
- Rejections and cancellations with required remarks.
- Queue dashboards, trend charts, status breakdowns, and recent activity.
- Server-sent events for workflow updates and notification refreshes.
- CSV export, filters, sorting, pagination, and printable truck labels.
- Password hashing and OTP support for FTE flows.

## Advanced Features

- Rate limiting on API requests.
- Cache headers for static assets.
- Dockerized backend/frontend and NGINX reverse proxy.
- Optional adapters for Clerk, Resend, PostHog, Sentry, Upstash, Pinecone, and Better Stack through environment variables.

## Success Measures

- Operators can complete the full truck-request flow without Google Sheets.
- API endpoints remain responsive with paginated/limited queries.
- Workflow updates appear without page refreshes.
- Secrets and deployment settings are externalized through environment variables.
