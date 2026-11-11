# Deploy — test environment first, real domain later

Everything is driven from GitHub; no local terminal is needed.

**Now (test): Render free + Supabase.** The Go API runs on Render's free plan from `render.yaml`; the site on
GitHub Pages calls it. **Later (production): Cloudflare** — section 1 below, already wired, same code.

## 0. Test environment on Render (current)

1. render.com → sign in with GitHub → **New → Blueprint** → repository `BrenoD/Celestial-Oud` (branch `frontend`).
2. It reads `render.yaml` and asks for **`DATABASE_URL`**: paste the Supabase transaction-pooler URI with the
   password. Everything else (JWT/cron secrets, CORS for GitHub Pages, `PAYMENTS_MODE=mock`) is preset.
3. Apply. First build takes a few minutes; the API then creates all tables in Supabase on boot.
4. Copy the service URL (e.g. `https://celestial-oud-api.onrender.com`).
5. GitHub → *Settings → Secrets and variables → Actions*:
   - **Variables** → `VITE_API_URL` = the Render URL.
   - **Secrets** → `CRON_SECRET` = the value Render generated (service → *Environment* → reveal `CRON_SECRET`).
6. GitHub → *Actions → Deploy preview to GitHub Pages → Run workflow*. The site now uses the real API.
7. First admin: section 2 below.

Free-plan notes: the service sleeps after ~15 min idle (first request then takes ~30–60 s); the *API cron tick*
workflow pings it every 10 min, which also keeps it mostly awake. Payments are simulated (`PAYMENTS_MODE=mock`):
orders are marked paid without charging. Pushes touching `backend/` redeploy automatically.

**Switching to Cloudflare later**: do section 1, then change `VITE_API_URL` to the Cloudflare URL (or delete it
to auto-detect `workers.dev`), copy `CRON_SECRET` accordingly, and delete the Render service.

```
GitHub push ──► deploy-api.yml ──► Cloudflare Worker + Container (Go API)
                                     https://celestial-oud-api.<subdomain>.workers.dev
                                          │  Supabase Postgres (eu-west-3, transaction pooler)
            └─► pages.yml (re-runs after each API deploy) ──► https://brenod.github.io/Celestial-Oud/
                 builds the site against the API above (falls back to the mock if it is down)
```

## 1. Cloudflare (production — ready, not active yet)

**Cloudflare** (dash.cloudflare.com)
1. Workers & Pages plan: **Workers Paid** (required for Containers).
2. Account ID: shown on the right of *Workers & Pages → Overview*.
3. API token: *My Profile → API Tokens → Create Token → "Edit Cloudflare Workers"* template, then also add
   **Account → Containers → Edit** and **Account → Cloudflare Images/Registry** if offered; scope it to your account.
4. *Workers & Pages → Overview*: make sure a `workers.dev` subdomain is claimed.

**GitHub** → repository *Settings → Secrets and variables → Actions → New repository secret*

| Secret | Value |
|---|---|
| `CLOUDFLARE_API_TOKEN` | token from step 3 |
| `CLOUDFLARE_ACCOUNT_ID` | account ID from step 2 |
| `DATABASE_URL` | Supabase transaction-pooler URI with the password, e.g. `postgresql://postgres.bualtfdrfvsnnydwtmgz:<password>@aws-1-eu-west-3.pooler.supabase.com:6543/postgres` (percent-encode special characters) |
| `JWT_SECRET` | any random string of 40+ characters (password manager → generate) |

Optional secrets: `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`, `RESEND_API_KEY`
(without it emails are only logged), `CRON_SECRET` (derived from `JWT_SECRET` when unset).

Payments: in production the API refuses checkout until Stripe is configured — unless the repository variable
`PAYMENTS_MODE=mock` is set (test only: orders are marked paid without charging).

Optional variable (*Variables* tab): `ADMIN_EMAILS` — comma-separated addresses that receive order notifications.

**Run it**: *Actions → Deploy API to Cloudflare → Run workflow*. The run summary prints the test URL. When it
finishes, *Deploy preview to GitHub Pages* re-runs by itself and the site at
https://brenod.github.io/Celestial-Oud/ uses the real API.

## 2. First admin

1. On the site: *Account → Create account* with your email.
2. Supabase → **SQL Editor**:
   ```sql
   update users set role = 'admin' where email = 'you@example.com';
   ```
3. Sign out and back in → *Account* shows **Admin →**. Invite the rest of the team from *Admin → Users*.

## 3. Production on celestialoud.com (Cloudflare)

| What | URL | Workflow |
|---|---|---|
| Storefront + admin | https://celestialoud.com (`www` → 301 to apex) | `deploy-site.yml` → Worker `celestial-oud-site` (`frontend/cloudflare/`, static assets, SPA fallback) |
| API | https://api.celestialoud.com | `deploy-api.yml` → Worker + Container `celestial-oud-api` (`backend/cloudflare/`) |

Both are custom domains declared in their `wrangler.toml`; Cloudflare creates the DNS records and certificates on
deploy. Requirements: the `celestialoud.com` zone on the same account, the Workers Paid plan (Containers), and the
repository secrets `CLOUDFLARE_API_TOKEN` (template *Edit Cloudflare Workers* + *Containers: Edit*, all zones),
`CLOUDFLARE_ACCOUNT_ID`, `DATABASE_URL`, `JWT_SECRET`.

Defaults need no repository variables. Overrides: `SITE_API_URL` (storefront build), `PUBLIC_URL`, `API_URL`,
`CORS_ORIGINS`, `ADMIN_EMAILS`, `PAYMENTS_MODE` (API). Without Stripe keys the production checkout is refused
(503) unless `PAYMENTS_MODE=mock`. Stripe webhook → `https://api.celestialoud.com/webhooks/stripe`.

GitHub Pages + Render stay up as the preview/fallback environment until they are switched off.

## Caching & load

Public catalogue responses carry `Cache-Control: public, max-age=0, must-revalidate, s-maxage=N`: the Worker
keeps them at the Cloudflare edge for N seconds (30 s for products/prices, 5 min for collections/content),
browsers always revalidate. Admin edits are visible to the editor immediately and to shoppers within N seconds.
