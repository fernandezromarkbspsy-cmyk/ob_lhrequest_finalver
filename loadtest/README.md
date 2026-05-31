# Load testing

This k6 script models the go-live risk called out for thousands of users: each virtual user signs in, polls `/api/stats` and paginated `/api/requests`, and opens `/api/events` to exercise SSE connection handling.

Example:

```bash
BASE_URL=https://app.example.com \
LOGIN_EMAIL=loadtest@example.com \
LOGIN_PASSWORD='replace-me' \
TARGET_VUS=1000 \
k6 run loadtest/sse-polling.k6.js
```

Run against a staging database with seeded data. Increase `TARGET_VUS` gradually (100, 500, 1000, 5000) while watching `/metrics`, Postgres connection usage, Redis latency, API CPU/memory, and NGINX connection counts.
