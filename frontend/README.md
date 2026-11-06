# Celestial Oud — storefront (frontend)

React 18 + TypeScript + Vite single-page app for the Celestial Oud perfume shop.
It is a **static build** served by nginx/CDN and talks to the Go API over JSON.
No server-side rendering, no server state: scaling to many simultaneous
visitors is a matter of adding nginx replicas or putting a CDN in front.

## Run

```bash
cd frontend
npm install
npm run dev          # http://localhost:5173 — uses the in-browser mock API
npm run build        # dist/ (typecheck + production bundle)
npm run preview      # serve dist/ locally
npm test             # vitest
npm run lint
```

Copy `.env.example` to `.env` and set `VITE_API_URL` to point at the Go API.
With it empty the app runs against `src/api/mock.ts` (placeholder catalogue in
`src/data/catalogue.ts`; prices/ratings are **placeholders**).

Mock-only helpers: discount code `WELCOME10` (10%), any UK postcode returns
three example addresses, any email/password signs in.

### Docker

```bash
docker build -t celestial-oud-storefront --build-arg VITE_API_URL=https://api.example.com ./frontend
docker run -p 8080:8080 celestial-oud-storefront
# or from the repo root:
docker compose up --build
```

`/healthz` returns 200 for load balancers. `nginx.conf` caches `/assets/*`
(content-hashed) for a year, never caches `index.html`, gzips text, and adds
basic security headers.

## How it scales

| Concern | Approach |
|---|---|
| Many concurrent visitors | Pure static files; nginx serves with `sendfile`; stateless so N replicas / CDN edge caching work without sticky sessions. |
| Bundle size | Route-level `React.lazy` chunks (home ≈ 7 kB gz + shared vendor). Vendor split into long-lived `react` and `query` chunks so app releases don't invalidate them. |
| API load | TanStack Query caches and de-duplicates requests (`staleTime` 1–5 min for catalogue), aborts stale list requests, `keepPreviousData` for filters. |
| Resilience | Fetch client has a 10 s timeout, exponential-backoff retries for idempotent GETs, and clears auth on 401. `ErrorBoundary` around the router. |
| Safe checkout retries | `POST /v1/orders` sends an `Idempotency-Key` (UUID per checkout attempt): a retried request after a network blip returns the same order instead of creating a duplicate. |
| Basket | Persisted in `localStorage` (Zustand `persist`) — no session store needed on the server. |
| Payments | Card fields are UI only. In production the PSP (Stripe/Adyen Elements) tokenises in the browser and only `paymentToken` is posted; raw card data must never hit the Go API. |

## Project layout

```
src/
  api/        client.ts (fetch wrapper), real.ts (Go API), mock.ts, hooks.ts (TanStack Query), index.ts (facade)
  components/ layout/ (announcement bar, header, nav, footer, cookie banner, toast) · ui/ (ProductCard, Accordion, Qty, Stars, Icons…)
  data/       catalogue.ts — placeholder data used by the mock
  lib/        money.ts (pence → £), consent.ts
  pages/      Home, Collection (+search), Product, Basket, Checkout, Confirmation, Account, Help, About, Wishlist, Quiz, Legal, NotFound
  store/      basket.ts, auth.ts, wishlist.ts (Zustand)
  styles/     global.css (design tokens + primitives), layout.css
  types.ts    shared domain types (keep in sync with the Go API)
```

## Routes

| Path | Page |
|---|---|
| `/` | Home |
| `/collections/:slug` | Listing with URL-driven filters (`?family=…&notes=…&size=…&price=…&wear=…&sort=…&page=…`) |
| `/search?q=` | Search results (same component) |
| `/products/:slug` | Product page |
| `/basket` | Basket (free-delivery bar, free sample picker, discount code, gift wrap) |
| `/checkout` | Checkout (distraction-free layout) |
| `/checkout/confirmed/:id` | Order confirmation |
| `/account`, `/account/:tab` | Sign in / register → dashboard (orders, wishlist, scent profile, addresses, preferences, privacy & data) |
| `/wishlist` | Wishlist (local) |
| `/quiz` | 5-question scent finder → filtered collection |
| `/help` | Track order, delivery, returns, FAQ, contact |
| `/about` | Our story |
| `/legal/:page` | terms · privacy · cookies · accessibility · product-safety |

## REST contract expected from the Go API

All money is integer pence. Errors: `{ "message": string, "code"?: string, "details"?: any }` with a 4xx/5xx status.

| Method | Path | Returns |
|---|---|---|
| GET | `/v1/collections` | `Collection[]` |
| GET | `/v1/collections/{slug}` | `Collection` |
| GET | `/v1/products?collection&q&family&notes&size&price&wear&sort&page&pageSize` (CSV arrays) | `Page<ProductSummary>` |
| GET | `/v1/products/{slug}` | `Product` |
| GET | `/v1/products/{slug}/related` | `ProductSummary[]` |
| GET | `/v1/products/{slug}/reviews` | `Review[]` |
| GET | `/v1/delivery-methods` | `DeliveryMethod[]` |
| GET | `/v1/address/lookup?postcode=` | `Address[]` (line1, line2, city, postcode) |
| POST | `/v1/discounts/validate` `{code, subtotal}` | `{code, amount}` |
| POST | `/v1/orders` + header `Idempotency-Key` | `Order` |
| GET | `/v1/orders/{id}?email=` | `Order` (guest lookup) |
| GET | `/v1/account/orders` (Bearer) | `Order[]` |
| POST | `/v1/auth/login`, `/v1/auth/register` | `{token, user}` |
| POST | `/v1/newsletter` `{email}` | 204 |
| POST | `/v1/contact` `{name,email,orderId?,message}` | 204 |
| GET | `/v1/content/faqs` | `Faq[]` |

Types live in `src/types.ts`. Samples use virtual SKUs `"<slug>-2ml"`.

## Still placeholder (`[XX]` in copy)

Prices, review counts, company number/address, policy text, support phone,
product descriptions and INCI lists, real photography, PSP integration,
social login, analytics (gated behind the cookie consent event `co:consent`).
