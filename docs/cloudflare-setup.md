# Cloudflare Setup Guide

This guide walks through configuring Cloudflare as the DNS manager, SSL provider, CDN, and DDoS protection layer for the SOC 5 Dashboard.

---

## 1. Add Your Domain to Cloudflare

1. Log in to [dash.cloudflare.com](https://dash.cloudflare.com)
2. Click **Add a Site**, enter your domain (e.g., `soc5dashboard.com`), and choose the **Free** plan or higher
3. Cloudflare will scan your existing DNS records
4. Update your domain's nameservers at your registrar to the two Cloudflare nameservers shown (e.g., `ada.ns.cloudflare.com`, `bob.ns.cloudflare.com`)
5. Wait for propagation (usually under 5 minutes)

---

## 2. DNS Records

Add these records in Cloudflare DNS:

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| `A` | `@` | `<your-server-IP>` | ✅ Proxied |
| `A` | `www` | `<your-server-IP>` | ✅ Proxied |
| `CNAME` | `api` | `@` | ✅ Proxied |

> Keep all records **Proxied** (orange cloud) so traffic flows through Cloudflare.

---

## 3. SSL/TLS

1. Go to **SSL/TLS** → **Overview**
2. Set mode to **Full (strict)**
   - "Full" encrypts between browser → Cloudflare and Cloudflare → your server
   - "Strict" requires a valid certificate on your server (use the Docker Nginx config with a self-signed cert, or install Let's Encrypt)
3. Under **Edge Certificates**, enable:
   - **Always Use HTTPS** ✅
   - **Automatic HTTPS Rewrites** ✅
   - **HSTS** — enable with `max-age=31536000` once you're confident HTTPS works

> If you don't want to manage a server certificate, use **Flexible** mode (browser→CF is HTTPS, CF→server is HTTP). Only do this in a private/internal network.

---

## 4. CDN & Caching

1. Go to **Caching** → **Configuration**
2. Set **Caching Level** to **Standard**
3. Set **Browser Cache TTL** to **4 hours**
4. Under **Cache Rules**, add a rule:
   - **URL**: `*.yourdomain.com/assets/*`
   - **Cache**: Edge Cache TTL = **1 year** (Vite hashes all asset filenames)
5. Add a second rule to bypass cache for API:
   - **URL**: `*.yourdomain.com/api/*`
   - **Cache**: Bypass

---

## 5. DDoS Protection

Cloudflare provides automatic DDoS protection on all plans. To tune it:

1. Go to **Security** → **DDoS**
2. Set **HTTP DDoS attack protection** to **High** sensitivity
3. Under **Security** → **WAF**, enable **Managed Rules** (available on Pro+)

For the Free plan, you still get:
- Automatic L3/L4 network DDoS mitigation
- Basic L7 HTTP flood protection
- Bot fight mode (enable under **Security** → **Bots**)

---

## 6. Page Rules / Rate Limiting

Add a rate limit rule on the login endpoint (Pro plan):

1. **Security** → **WAF** → **Rate limiting rules**
2. **Field**: URI Path equals `/api/login`
3. **Requests**: more than 10 in 60 seconds per IP
4. **Action**: Block for 5 minutes

---

## 7. Environment Variable

Once your domain is live, set `APP_SECRET` in your `.env` file to a strong random value:

```bash
openssl rand -hex 32
```

Then add to `.env`:
```env
APP_SECRET=<generated-value>
```

---

## Architecture Overview

```
Internet → Cloudflare (DNS + SSL + DDoS + CDN)
              ↓ HTTPS
         NGINX (port 80, Cloudflare terminates TLS)
         ├── /assets/* → React SPA static files (cached 1yr)
         ├── /api/*    → Go API server (port 8080)
         └── /*        → React SPA index.html
              ↓
         Redis (cache + SSE pub/sub)
              ↓
         PostgreSQL (Supabase)
```
