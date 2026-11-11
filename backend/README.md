# Celestial Oud — API (Go)

Single static binary: chi router, pgx against **Supabase Postgres**, embedded SQL migrations applied at
start-up, own JWT auth, Stripe PaymentIntents, Resend email. Deployed as a **Cloudflare Container** behind a
Worker that adds edge caching and a 5-minute cron. Data model: [`../docs/DATA-MODEL.md`](../docs/DATA-MODEL.md).

## Run locally

```bash
cd backend
cp .env.example .env            # set DATABASE_URL (any Postgres 14+)
go run ./cmd/api                # migrations run automatically → http://localhost:8080
go test ./... && go vet ./...
```

Without `STRIPE_SECRET_KEY` the checkout runs in **mock payment mode** in development (orders are marked paid at once); with `APP_ENV=production` it is refused unless `PAYMENTS_MODE=mock` is set explicitly;
without `RESEND_API_KEY` emails are logged instead of sent. Point the storefront at it with
`VITE_API_URL=http://localhost:8080`.

### Supabase (production database)

Project `bualtfdrfvsnnydwtmgz`, region **West EU (Paris, eu-west-3)**, plan Nano (60 direct connections).
Use the **Transaction pooler** URI from *Project Overview → Get connected → Direct → Connection string*:
`postgresql://postgres.bualtfdrfvsnnydwtmgz:<password>@aws-1-eu-west-3.pooler.supabase.com:6543/postgres`.
The API detects the pooler and switches pgx to unnamed statements (`QueryExecModeDescribeExec`, no
statement cache); migrations use transaction-scoped advisory locks, so any number of instances can boot at
once (tested with 4 concurrent boots × 5 rounds). The first boot creates every table; nothing has to be run in
the Supabase SQL editor. The publishable/anon key and Supabase Auth are **not** used — the API talks to Postgres
directly and issues its own JWTs.

First admin: register a user through the storefront, then
`update users set role = 'admin' where email = '<you>';` — further staff are invited from the admin UI.

## Layout

```
cmd/api/main.go            boot: config → db + migrate → router → graceful shutdown
internal/config            env configuration
internal/db                pgx pool + embedded migrations (internal/db/migrations/*.sql)
internal/httpx             JSON helpers, API error type, request-id/logging/recover/security/rate-limit middleware, role ranks
internal/auth              bcrypt, HS256 access tokens (15 min), rotating opaque refresh tokens (30 d), Require/RequireRole
internal/mail              Resend client; templates gated by email_templates / admin_notifications rows
internal/store             data access (users, catalog, orders, engagement, admin, analytics, cron jobs)
internal/api               handlers: catalog, auth, account, orders (+Stripe, webhook, cron), admin, payments
cloudflare/                Worker (TypeScript) + wrangler.toml for Containers
Dockerfile                 distroless static image (~10 MB)
```

## Routes

Public: `GET /v1/collections[/{slug}]`, `GET /v1/products` (filters: `collection,q,family,notes,size,price,wear,sort,page,pageSize`),
`GET /v1/products/{slug}[/related|/reviews]`, `POST /v1/products/{slug}/reviews` (auth), `POST /v1/products/{slug}/notify`,
`GET /v1/notes`, `GET /v1/delivery-methods`, `GET /v1/address/lookup?postcode=`, `POST /v1/discounts/validate`,
`GET /v1/content/faqs|{key}`, `GET /v1/blog[/{slug}]`, `GET /v1/settings`, `POST /v1/newsletter[/unsubscribe]`, `POST /v1/contact`.

Orders: `POST /v1/orders` (**`Idempotency-Key` required**; returns the order and, with Stripe, `payment.clientSecret`),
`GET /v1/orders/{id}?email=`, `POST /v1/orders/{id}/confirm`.

Auth: `POST /v1/auth/register|login|refresh|logout|forgot|reset`. Account (Bearer): `GET|PATCH|DELETE /v1/account/`,
`POST /v1/account/password`, `GET /v1/account/export` (GDPR), `GET /v1/account/orders`, `GET|POST|PUT|DELETE /v1/account/addresses`,
`GET /v1/account/wishlist`, `POST /v1/account/wishlist/{slug}`, `GET|PUT /v1/account/scent-profile`.

Admin (`/v1/admin/*`, role ≥ viewer; writes gated support/manager/admin — see `internal/api/admin.go`):
dashboard, analytics, audit, products (+stock), collections, stock-alerts, orders (status, tracking, refund), customers, payments,
discounts, promotions, reviews, newsletter (+send), messages (+reply), delivery-methods, content, blog, sections/{taxes|emails|notifications|integrations},
settings, users (invite, role, reset-password).

System: `POST /webhooks/stripe` (signature-verified, idempotent by event id), `POST /internal/cron/tick` (`X-Cron-Secret`),
`GET /healthz`, `GET /readyz`.

## Order lifecycle

`pending_payment` → (`paid`) → `processing` → `dispatched` → `delivered`; `cancelled` / `refunded` restock items.
Stock is reserved when the order is created; unpaid orders expire after 24 h (cron). The Stripe webhook and
`POST /orders/{id}/confirm` both call `MarkPaid`, which is idempotent, so emails and loyalty points fire exactly once.

## Deploy to Cloudflare

Recommended: from GitHub Actions, no terminal needed — see [`docs/DEPLOY.md`](../docs/DEPLOY.md)
(`.github/workflows/deploy-api.yml` publishes to a free `workers.dev` test URL). Manual alternative:

```bash
cd backend/cloudflare && npm install
npx wrangler secret put DATABASE_URL      # Supabase pooler URL (or use Hyperdrive)
npx wrangler secret put JWT_SECRET
npx wrangler secret put CRON_SECRET
npx wrangler secret put STRIPE_SECRET_KEY STRIPE_PUBLISHABLE_KEY STRIPE_WEBHOOK_SECRET RESEND_API_KEY ADMIN_EMAILS
npx wrangler deploy                        # builds ../Dockerfile, pushes the image, deploys Worker + Container
```

Requires the Workers Paid plan. Then set the Stripe webhook to `https://<api-host>/webhooks/stripe`
(`payment_intent.succeeded`, `payment_intent.payment_failed`, `charge.refunded`) and point the storefront's
`VITE_API_URL` at the Worker URL. Scale by raising `max_instances` / `INSTANCES` in `wrangler.toml` / `src/index.ts`.
