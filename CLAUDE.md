# Celestial Oud — repo guide

Read `handoff/HANDOFF.md` first: it holds the brand palette, UX principles, UK
legal notes and the open client questions. Everything in `[XX]` is placeholder
data still to be confirmed by the client.

## Layout
- `frontend/` — storefront SPA (React 18, TypeScript, Vite, React Router, TanStack Query, Zustand). See `frontend/README.md` for scripts, routes and the REST contract.
- `handoff/` — design mockups (`design/*.dc.html`, proprietary format, reference only) and assets.
- Backend (Go, PostgreSQL, JWT) and admin UI: not started; keep `frontend/src/types.ts` and the contract table in `frontend/README.md` in sync when you add endpoints.

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
