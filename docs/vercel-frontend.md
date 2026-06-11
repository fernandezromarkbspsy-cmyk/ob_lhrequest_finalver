# Vercel Frontend Deployment Guide

This guide deploys only the Gentelella-inspired vanilla JS browser frontend from `frontend/` to Vercel.

Use this with either:

- AWS EC2 backend from `docs/aws-deployment.md`
- Vercel backend from `docs/deployment.md`
- Any other backend that exposes the Go API over HTTPS

The frontend is not a Next.js, React, or Vite app. It is a Gentelella-inspired vanilla JS static app: HTML entry pages, `frontend/static/gentelella.css`, `frontend/static/app.css`, `frontend/static/app.js`, and image assets.

The HTML files are still required because they are the browser entry points for each page. Gentelella is the admin UI theme layer, and vanilla JavaScript in `app.js` provides the behavior.

## Required Values

Prepare these before deploying:

```text
BACKEND_URL=https://api.your-domain.com
FRONTEND_URL=https://your-app.vercel.app
```

Examples:

```text
BACKEND_URL=https://api.soc5.example.com
FRONTEND_URL=https://soc5-dashboard.vercel.app
```

`BACKEND_URL` is used in `frontend/config.js`.

`FRONTEND_URL` is used on the backend as `FRONTEND_URL` for CORS.

## 1. Configure the Frontend API URL

Edit:

```text
frontend/config.js
```

For production deployment, set `API_BASE` to your backend origin:

```js
(function () {
  window.SOC5_CONFIG = {
    API_BASE: "https://api.your-domain.com"
  };
})();
```

Important:

- Include `https://`.
- Do not add a trailing slash.
- Do not include a path like `/api`.
- The backend already has routes like `/api/login`, `/api/requests`, and `/healthz`.

If you are still testing locally, keep the current local-aware config:

```js
(function () {
  var isLocalFrontend = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1";
  var isSeparateLocalPort = isLocalFrontend && window.location.port && window.location.port !== "8080";

  window.SOC5_CONFIG = {
    API_BASE: isSeparateLocalPort ? "http://localhost:8080" : ""
  };
})();
```

Before a real Vercel production deploy, switch it to the deployed backend URL.

## 2. Import the Project in Vercel

1. Open Vercel.
2. Choose Add New Project.
3. Import the Git repository.
4. In project settings, set:

```text
Framework Preset: Other
Root Directory: frontend
Build Command: leave empty
Output Directory: leave empty or .
Install Command: leave empty
```

Vercel will serve the Gentelella-inspired static frontend files from `frontend/`.

## 3. Deploy

Click Deploy.

After deployment, Vercel will show a URL like:

```text
https://your-app.vercel.app
```

Open:

```text
https://your-app.vercel.app/dashboard.html
```

Other app pages:

```text
https://your-app.vercel.app/lh-request.html
https://your-app.vercel.app/truck-request.html
https://your-app.vercel.app/dock-officer.html
https://your-app.vercel.app/settings.html
```

## 4. Configure Backend CORS

On the backend host, set:

```env
FRONTEND_URL=https://your-app.vercel.app
```

For AWS EC2, edit:

```bash
sudo nano /etc/soc5-dashboard.env
```

Then restart:

```bash
sudo systemctl restart soc5-dashboard
```

For a Vercel-hosted backend, add `FRONTEND_URL` in Vercel Project Settings under Environment Variables.

The value must exactly match the frontend origin:

```text
https://your-app.vercel.app
```

Do not use:

```text
https://your-app.vercel.app/
https://your-app.vercel.app/dashboard.html
http://your-app.vercel.app
```

## 5. Verify Browser Requests

Open the Vercel frontend in the browser.

Open DevTools, then Network.

Confirm:

- `frontend/config.js` loads.
- API calls go to `https://api.your-domain.com`.
- `GET /healthz` returns `200`.
- `POST /api/login` reaches the backend.
- No request is blocked by CORS.

You can also test the backend directly:

```powershell
curl https://api.your-domain.com/healthz
```

Expected:

```json
{"ok":true}
```

## 6. Custom Domain

In Vercel:

1. Open the frontend project.
2. Go to Settings.
3. Open Domains.
4. Add your frontend domain, for example:

```text
dashboard.your-domain.com
```

5. Follow the DNS instructions shown by Vercel.

After the custom domain works, update backend CORS again:

```env
FRONTEND_URL=https://dashboard.your-domain.com
```

Restart or redeploy the backend after changing this value.

## 7. Updating the Frontend

After editing frontend files:

1. Commit and push changes.
2. Vercel automatically creates a preview deployment.
3. Promote or merge to production when verified.

If you change `frontend/config.js`, confirm the deployed browser file has the expected backend URL:

```text
https://your-app.vercel.app/config.js
```

## Troubleshooting

### Page loads but API data is empty

Check `frontend/config.js`. `API_BASE` probably still points to `""`, which means the browser is calling the Vercel frontend domain instead of the backend.

Set:

```js
API_BASE: "https://api.your-domain.com"
```

Redeploy the frontend.

### CORS error in browser devtools

The backend `FRONTEND_URL` does not exactly match the Vercel frontend origin.

Set backend env:

```env
FRONTEND_URL=https://your-app.vercel.app
```

Restart or redeploy the backend.

### Vercel shows a build error

The frontend has no build step because it is already vanilla JS plus static Gentelella-inspired assets.

Use:

```text
Framework Preset: Other
Root Directory: frontend
Build Command: empty
Output Directory: empty or .
Install Command: empty
```

### Direct page URL returns 404

Use the actual HTML filename:

```text
/dashboard.html
/lh-request.html
/truck-request.html
/dock-officer.html
/settings.html
```

This app is a multi-page static app, not a single-page router app.

## Official References

- Vercel build configuration: https://vercel.com/docs/builds/configure-a-build
- Vercel project configuration: https://vercel.com/docs/project-configuration
- Vercel supported frameworks: https://vercel.com/docs/frameworks
