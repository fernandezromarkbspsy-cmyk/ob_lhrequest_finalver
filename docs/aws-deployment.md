# AWS EC2 Backend Deployment Guide

This guide deploys the SOC 5 outbound dashboard backend to AWS EC2 and keeps the frontend as a static site on Vercel, Netlify, Cloudflare Pages, or another static host.

Recommended production layout:

- Frontend: static files from `frontend/`
- Backend: Go/Echo API from `cmd/server` on AWS EC2
- Database: existing Supabase Postgres
- Reverse proxy: Nginx on EC2
- Process manager: systemd
- HTTPS: Certbot/Let's Encrypt

This setup is a better fit than serverless hosting for this backend because `/api/events` uses server-sent events. EC2 runs the Go server as a normal long-running process.

## Cost Notes

For a small deployment, start with one EC2 instance.

Recommended instance:

```text
t4g.small
```

Notes:

- AWS has offered `t4g.small` trial/free-tier capacity in many regions, but the exact free-tier behavior depends on account age, region, and AWS plan.
- Create an AWS Budget before deploying.
- Keep Supabase as the database first. Moving Postgres to RDS adds cost and setup work.
- Avoid App Runner if the goal is lowest monthly cost. It is simpler than EC2, but a continuously warm API usually costs more than a tiny EC2 instance.

## Required Values

Prepare these values before starting:

```text
AWS_REGION=ap-southeast-1
DOMAIN_API=api.your-domain.com
FRONTEND_URL=https://your-frontend.example.com
DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
```

If you do not have a frontend deployment yet, you can deploy the backend first and update `FRONTEND_URL` later.

## 1. Launch EC2

In the AWS Console:

1. Open EC2.
2. Choose Launch Instance.
3. Name it:

```text
soc5-dashboard-backend
```

4. Choose an AMI:

```text
Ubuntu Server 24.04 LTS
```

5. Choose architecture:

```text
64-bit Arm
```

6. Choose instance type:

```text
t4g.small
```

7. Create or select a key pair.
8. Configure storage:

```text
16 GB gp3
```

9. Create a security group with these inbound rules:

```text
SSH    TCP 22   your public IP only
HTTP   TCP 80   0.0.0.0/0
HTTPS  TCP 443  0.0.0.0/0
```

Do not expose Postgres to the internet. The app connects outward to Supabase.

## 2. Connect to EC2

From your local machine:

```powershell
ssh -i .\your-key.pem ubuntu@your-ec2-public-ip
```

On Windows, the key file may need restricted permissions. If SSH rejects the key, run:

```powershell
icacls .\your-key.pem /inheritance:r
icacls .\your-key.pem /grant:r "$env:USERNAME:R"
```

Then try SSH again.

## 3. Install Server Packages

On the EC2 instance:

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y git curl nginx certbot python3-certbot-nginx
```

## 4. Install Go

This project requires Go 1.25 or newer.

Check Ubuntu's available Go version:

```bash
apt-cache policy golang-go
```

If it is Go 1.25 or newer, install it:

```bash
sudo apt install -y golang-go
go version
```

If Ubuntu provides an older Go version, install Go from the official tarball instead:

```bash
cd /tmp
curl -LO https://go.dev/dl/go1.25.0.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.0.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
go version
```

For x86 EC2 instances, use the `linux-amd64` Go tarball instead of `linux-arm64`.

## 5. Deploy the Backend Code

Create an application directory:

```bash
sudo mkdir -p /opt/soc5-dashboard
sudo chown ubuntu:ubuntu /opt/soc5-dashboard
```

Clone the repository:

```bash
cd /opt/soc5-dashboard
git clone https://github.com/your-owner/your-repo.git app
cd app
```

If the repository is private, either configure SSH deploy keys or upload a zip from your machine.

Build the backend:

```bash
go mod download
go build -o /opt/soc5-dashboard/soc5-backend ./cmd/server
```

## 6. Configure Backend Environment

Create an environment file outside the Git repository:

```bash
sudo nano /etc/soc5-dashboard.env
```

Use this template:

```env
APP_ENV=production
APP_HOST=127.0.0.1
APP_PORT=8080
FRONTEND_URL=https://your-frontend.example.com
DATABASE_URL=postgres://your-user:your-password@your-host:5432/your-database?sslmode=require
```

Important:

- `APP_HOST=127.0.0.1` is intentional when using Nginx. The API is reachable publicly through Nginx, not directly on port `8080`.
- `FRONTEND_URL` must be the exact frontend origin in the browser, for example `https://your-app.vercel.app`.
- Do not add a trailing slash or path to `FRONTEND_URL`.
- Do not put `.env` or production secrets in Git.

Lock down the environment file:

```bash
sudo chown root:root /etc/soc5-dashboard.env
sudo chmod 600 /etc/soc5-dashboard.env
```

## 7. Create a systemd Service

Create the service file:

```bash
sudo nano /etc/systemd/system/soc5-dashboard.service
```

Paste:

```ini
[Unit]
Description=SOC 5 Dashboard Go Backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/opt/soc5-dashboard/app
EnvironmentFile=/etc/soc5-dashboard.env
ExecStart=/opt/soc5-dashboard/soc5-backend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable soc5-dashboard
sudo systemctl start soc5-dashboard
```

Check status:

```bash
sudo systemctl status soc5-dashboard --no-pager
```

Check logs:

```bash
journalctl -u soc5-dashboard -f
```

Test locally on EC2:

```bash
curl http://127.0.0.1:8080/healthz
```

Expected result:

```json
{"ok":true}
```

## 8. Configure Nginx Reverse Proxy

Create an Nginx site:

```bash
sudo nano /etc/nginx/sites-available/soc5-dashboard
```

Paste this config and replace `api.your-domain.com`:

```nginx
server {
    listen 80;
    server_name api.your-domain.com;

    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/events {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 1h;
    }
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/soc5-dashboard /etc/nginx/sites-enabled/soc5-dashboard
sudo nginx -t
sudo systemctl reload nginx
```

## 9. Point DNS to EC2

In your DNS provider, create this record:

```text
Type: A
Name: api
Value: your EC2 public IPv4 address
TTL: automatic or 300
```

Wait for DNS to resolve:

```bash
dig api.your-domain.com
```

From your local machine, test:

```powershell
curl http://api.your-domain.com/healthz
```

## 10. Enable HTTPS

After DNS points to EC2, run:

```bash
sudo certbot --nginx -d api.your-domain.com
```

Choose the redirect-to-HTTPS option when prompted.

Test renewal:

```bash
sudo certbot renew --dry-run
```

Test the backend:

```bash
curl https://api.your-domain.com/healthz
```

## 11. Configure the Frontend

Edit:

```text
frontend/config.js
```

For a deployed frontend, set the backend API origin:

```js
(function () {
  window.SOC5_CONFIG = {
    API_BASE: "https://api.your-domain.com"
  };
})();
```

Deploy the `frontend/` directory to your static host.

For Vercel:

1. Create a new Vercel project.
2. Import the repository.
3. Set the project root or output/static directory to:

```text
frontend
```

4. Leave build command empty unless Vercel requires one.
5. Deploy.

After deployment, copy the frontend URL shown by Vercel, for example:

```text
https://your-app.vercel.app
```

Update the EC2 backend environment:

```bash
sudo nano /etc/soc5-dashboard.env
```

Set:

```env
FRONTEND_URL=https://your-app.vercel.app
```

Restart the backend:

```bash
sudo systemctl restart soc5-dashboard
```

## 12. Verify the Full App

Check the backend:

```powershell
curl https://api.your-domain.com/healthz
```

Open the frontend:

```text
https://your-app.vercel.app/dashboard.html
```

In browser devtools, confirm:

- API calls go to `https://api.your-domain.com`.
- There are no CORS errors.
- Login reaches `POST /api/login`.
- Request data reaches `GET /api/requests`.
- Events connect to `GET /api/events`, or polling fallback continues to work.

## 13. Deploy Updates

SSH into EC2:

```powershell
ssh -i .\your-key.pem ubuntu@your-ec2-public-ip
```

Pull and rebuild:

```bash
cd /opt/soc5-dashboard/app
git pull
go mod download
go build -o /opt/soc5-dashboard/soc5-backend ./cmd/server
sudo systemctl restart soc5-dashboard
sudo systemctl status soc5-dashboard --no-pager
```

Check logs if needed:

```bash
journalctl -u soc5-dashboard -n 100 --no-pager
```

## 14. Rollback

Find recent commits:

```bash
cd /opt/soc5-dashboard/app
git log --oneline -n 10
```

Rollback to a known good commit:

```bash
git checkout GOOD_COMMIT_SHA
go build -o /opt/soc5-dashboard/soc5-backend ./cmd/server
sudo systemctl restart soc5-dashboard
```

To return to the main branch later:

```bash
git checkout main
git pull
go build -o /opt/soc5-dashboard/soc5-backend ./cmd/server
sudo systemctl restart soc5-dashboard
```

## 15. Security Checklist

- Restrict SSH to your public IP.
- Keep ports `80` and `443` public.
- Do not open port `8080` publicly when using Nginx.
- Store production env vars only in `/etc/soc5-dashboard.env`.
- Do not commit `.env`.
- Keep `DATABASE_URL` private.
- Use `sslmode=require` for Supabase.
- Set an AWS Budget and billing alert.
- Regularly run:

```bash
sudo apt update
sudo apt upgrade -y
```

## Troubleshooting

### Backend returns CORS errors

Check:

```bash
sudo cat /etc/soc5-dashboard.env
```

`FRONTEND_URL` must exactly match the frontend origin:

```text
https://your-app.vercel.app
```

Then restart:

```bash
sudo systemctl restart soc5-dashboard
```

### Backend is down

Check service status:

```bash
sudo systemctl status soc5-dashboard --no-pager
journalctl -u soc5-dashboard -n 100 --no-pager
```

### Nginx returns 502

The Go backend is probably not running or not listening on `127.0.0.1:8080`.

Run:

```bash
curl http://127.0.0.1:8080/healthz
sudo systemctl status soc5-dashboard --no-pager
```

### HTTPS certificate fails

Confirm DNS points to the EC2 public IP:

```bash
dig api.your-domain.com
```

Confirm inbound security group allows port `80`.

### Database connection fails

Confirm `DATABASE_URL` includes SSL:

```text
sslmode=require
```

Confirm the Supabase database password is URL-safe. If the password has special characters, use the connection string copied directly from Supabase.

### Frontend calls the wrong API

Check:

```text
frontend/config.js
```

For deployed frontend, `API_BASE` must be:

```js
API_BASE: "https://api.your-domain.com"
```

Redeploy the frontend after changing this file.

