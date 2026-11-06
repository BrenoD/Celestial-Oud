# Celestial Oud — online store

UK perfume brand (oud / amber / musk Eau de Parfum). Built by Kartus.

| Folder | What |
|---|---|
| [`frontend/`](frontend/README.md) | Storefront — React + TypeScript + Vite static SPA, nginx Dockerfile. |
| [`handoff/`](handoff/HANDOFF.md) | Design handoff: brand decisions, legal notes, page map, `.dc.html` mockups and image assets. Reference only, not production code. |
| `docker-compose.yml` | Runs the storefront on http://localhost:8080. |
| `.github/workflows/frontend.yml` | CI: typecheck, lint, test, build, Docker smoke test. |

Backend (Go + PostgreSQL) and admin are not in this repo yet; the storefront
documents the REST contract it expects in `frontend/README.md`.

Quick start:

```bash
cd frontend && npm install && npm run dev
```
