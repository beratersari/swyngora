# Epic K: Mobile category discovery

**Priority:** P1 (mobile track — after Home dashboard)  
**Status:** done  
**Depends on:** Epic D Markets + Epic J Home (done)  
**Branch:** `feature/mobile-category-discovery` (from latest `develop`)  
**Design plan:** `docs/design/mobile-category-discovery.md`  
**Feature:** `docs/features/mobile-category-discovery.md`  
**Analysis:** `project-management/tasks/mobile/category-discovery/MCAT-A.md`  
**Tasks folder:** `project-management/tasks/mobile/category-discovery/`

## Goal

Let mobile users **browse markets by product-catalog category** (Meme, AI, defi, Layer1_Layer2, …) as a first-class discovery path — not only buried in Markets filters. Reuse existing `GET /tags` + `GET /spot?tag=`. Atomic Design + View + ViewModel; primary runtime **Chrome** (`npm run web`).

## APIs (existing backend — no new endpoints)

| UI | Path |
|----|------|
| Category list | `GET /api/v1/market/tags` |
| Markets in category | `GET /api/v1/market/spot` with `tag` / `tags` |
| Exchanges / sort / quote | Existing markets endpoints + `MarketsContext` |

Tag catalog is Binance-sourced; spot rows may carry tags cross-venue by base (server enrichment).

## Tasks

- [x] MCAT-A — Tags + spot tag-filter field matrix (analysis)  
- [x] MCAT-1 — Constants + category query / label helpers + tests  
- [x] MCAT-2 — Atomic category UI (grid / strip)  
- [x] MCAT-3 — Categories browse page (ViewModel + View + route)  
- [x] MCAT-4 — Apply tag to Markets list (context + navigation)  
- [x] MCAT-5 — Home featured strip + Markets entry points  
- [x] MCAT-6 — Loading / empty / error / i18n  
- [x] MCAT-7 — Tests polish + docs / board / changelog closeout  

## Acceptance

- User reaches a tag-filtered market list without opening the full filter form first  
- Home shows featured category chips (only tags present in live catalog)  
- Categories page supports search + pick any tag  
- Selecting a tag sets Markets `selectedTags` and shows filtered spot list  
- Empty tags / empty markets / error states handled  
- Polling / AppState behavior unchanged for Markets list  
- No `modules/*/components`; RTK only under `libs/api`  
- Tests green; design/feature/board/changelog updated  

## Out of scope

- New backend endpoints or OpenAPI changes  
- Multi-tag discovery UX (Filters remains multi-select)  
- Alerts, auth, paper trading  
- Web (`frontend/`) category UI  
- Server-driven featured taxonomy  

## MR grouping

`(A+1)` · `(2–3)` · `(4–5)` · `(6–7)`
