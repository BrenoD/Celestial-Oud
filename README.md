# Celestial Oud — online store

UK perfume brand (oud / amber / musk Eau de Parfum). Built by Kartus.

| Folder | What |
|---|---|
| [`frontend/`](frontend/README.md) | Storefront **and admin** (`/admin`) — React + TypeScript + Vite static SPA, nginx Dockerfile. |
| [`backend/`](backend/README.md) | Go API for Cloudflare Containers + Supabase Postgres (auth, catalogue, orders, Stripe, admin). |
| [`docs/`](docs/) | `DATA-MODEL.md` (printforge re-reading, three questions per entity), `PAGES.md` (page inventory). |
| [`handoff/`](handoff/HANDOFF.md) | Design handoff: brand decisions, legal notes, page map, `.dc.html` mockups and image assets. Reference only, not production code. |
| `docker-compose.yml` | Runs the storefront on http://localhost:8080. |
| `.github/workflows/frontend.yml` | CI: typecheck, lint, test, build, Docker smoke test. |

Quick start:

```bash
cd frontend && npm install && npm run dev        # storefront + admin on the in-browser mock
cd backend && cp .env.example .env && go run ./cmd/api   # real API; then VITE_API_URL=http://localhost:8080
```

Admin preview on GitHub Pages: https://brenod.github.io/Celestial-Oud/admin — sign in with an email containing
`admin`, `manager`, `support` or `viewer` (any password ≥ 8 chars); changes persist in your browser only.
