# Epic J: Mobile home dashboard

**Priority:** P1 (mobile track — next after AI chat)  
**Status:** todo  
**Depends on:** Epics C–I (markets, detail, favorites, pumps, batch RSI, AI chat) — done  
**Branch:** `feature/mobile-home-dashboard` (from latest `develop` after AI chat merges, or stacked)  
**Design plan:** `docs/design/mobile-home-dashboard.md`  
**Feature:** `docs/features/mobile-home-dashboard.md`  
**Tasks folder:** `project-management/tasks/mobile/home/`

## Goal

Replace the scaffold **Home** screen with a **live market dashboard**: favorites snapshot, top movers, high volume, pump teaser, and quick entry to Ask — using existing APIs only. Atomic Design + View + ViewModel; primary runtime **Chrome** (`npm run web`).

## APIs (existing backend — no new endpoints)

| Widget | Path |
|--------|------|
| Favorites strip | Watchlist context + `GET /ticker/24h` and/or batch indicators |
| Top movers | `GET /api/v1/market/spot?sort=priceChangePercent&order=desc&limit=N` |
| Top volume | `GET /api/v1/market/spot?sort=quoteVolume&order=desc&limit=N` |
| Pump teaser | `GET /api/v1/market/pumps/scan` (small `symbolLimit`) |
| Health (footer) | `GET /health` (optional, demoted) |
| Ask chip | Navigate to Ask tab (no API) |

Default exchange: **binance**; quote **USDT** (configurable later).

## Tasks

- [ ] MHOME-1 — Dashboard constants + spot/pump query helpers  
- [ ] MHOME-2 — Atomic dashboard UI (section header, mover row, teaser cards)  
- [ ] MHOME-3 — HomePage ViewModel (parallel RTK, poll, AppState pause)  
- [ ] MHOME-4 — HomePage View + pull-to-refresh + deep links  
- [ ] MHOME-5 — Empty / partial failure / loading UX  
- [ ] MHOME-6 — Tests (helpers + ViewModel + page)  
- [ ] MHOME-7 — Docs + board + changelog closeout  

## Acceptance

- Home shows at least movers + volume + pumps teaser without opening Markets  
- Favorites strip when user has stars; empty CTA otherwise  
- Tap row → Coin detail (correct exchange/symbol)  
- Pump teaser → Pumps tab or detail  
- Ask chip → Ask tab (optional draft)  
- Polling pauses when Home unfocused / app backgrounded  
- No `modules/*/components`; RTK only under `libs/api`  
- Tests green; docs updated  

## Out of scope

- New backend endpoints or alert engine  
- Cross-exchange compare (follow-up)  
- Editable widget layout / drag-and-drop  
- Web (`frontend/`) home redesign  
- Streaming AI on Home  

## MR grouping

`(1–2)` · `(3–4)` · `(5–6)` · `(7)`
