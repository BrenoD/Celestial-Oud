# Celestial Oud — repo guide

Read `handoff/HANDOFF.md` first: it holds the brand palette, UX principles, UK
legal notes and the open client questions. Everything in `[XX]` is placeholder
data still to be confirmed by the client.

## Layout
- `frontend/` — storefront SPA + admin at `/admin` (React 18, TypeScript, Vite, React Router, TanStack Query, Zustand). `src/admin/` holds the back office: `rbac.ts` (roles & nav groups), `api.real.ts` / `api.mock.ts` (same `AdminApi` interface), `components/ListManager.tsx` (config-driven CRUD), `sections/`.
- `backend/` — Go API (chi + pgx) for Cloudflare Containers against Supabase Postgres; migrations in `internal/db/migrations`. See `backend/README.md`.
- `docs/DATA-MODEL.md` — every entity with the three questions (DB? schedule vs plain table? FK reuse?). Update it when you add a table.
- `handoff/` — design mockups (`design/*.dc.html`, proprietary format, reference only) and assets.
- Keep `frontend/src/types.ts`, `frontend/src/admin/types.ts` and `backend/internal/store/types.go` in sync (camelCase JSON, integer pence).

## Rules that come from the client
- Background is white. Espresso `#2E1D15` for primary CTAs, bronze `#B98A5A` for promo/secondary. Gold text on light backgrounds uses `#7A5530`.
- Fonts: Cinzel (wordmark), Cormorant Garamond 600 (headings), Jost (UI).
- Never reference other brands ("smells like", "dupe"). UK trademark risk.
- Every product page shows ingredients/allergens; opened fragrances excluded from returns, stated before purchase.
- Cookie banner must offer reject with equal weight (UK GDPR / PECR).
- Copy is UK English.

## Working on the frontend
```bash
cd frontend && npm install
npm run dev | build | test | lint | typecheck
```
- Money is integer pence everywhere; format with `lib/money.ts`.
- Pages fetch through `api/hooks.ts` only; swap mock ↔ real API via `VITE_API_URL`.
- Keep routes lazy (`App.tsx`) and avoid defining components inside components (remounts inputs).
- Run the build before committing; CI runs typecheck, lint, vitest and a Docker smoke test.

## Working on the backend
```bash
cd backend && cp .env.example .env && go run ./cmd/api   # needs DATABASE_URL
go test ./... && go vet ./...
```
- Roles: customer < viewer < support < manager < admin (`httpx.HasRole`); gate admin routes with `auth.RequireRole`.
- `POST /v1/orders` must stay idempotent (`Idempotency-Key`); `MarkPaid` must stay idempotent (webhook + confirm both call it).
- Never trust client prices: `store.PriceOrder` recomputes everything from SKUs.
- No Stripe key → mock payment mode; no Resend key → emails are logged.
