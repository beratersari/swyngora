# Epic F: Mobile watchlist

**Priority:** P0 (mobile track — next after coin detail)  
**Status:** done  
**Depends on:** Epic D (markets) + Epic E (coin detail) — done  
**Branch:** `feature/mobile-watchlist`  
**Design plan:** `docs/design/mobile-watchlist.md`  
**Feature:** `docs/features/mobile-watchlist.md`  
**Tasks folder:** `project-management/tasks/mobile/watchlist/`

## Goal

Let mobile users **save and manage spot pairs** (multi-exchange) with stars on Markets/Detail and a dedicated **Watchlist** tab. Use existing backend watchlist CRUD + ticker enrichment. **Atomic Design** + **View + ViewModel**; primary runtime **Chrome** (`npm run web`).

## APIs (existing backend)

| UI | Path |
|----|------|
| Get list | `GET /api/v1/watchlist` |
| Add | `POST /api/v1/watchlist/items` |
| Remove | `DELETE /api/v1/watchlist/items` |
| Replace | `PUT /api/v1/watchlist` |
| Quotes | `GET /api/v1/market/ticker/24h` |

Constraints: max **200** items; non-empty clientId ≠ `default`; store is in-memory.

## Tasks

- [x] MWL-1 — clientId + local storage helpers  
- [x] MWL-2 — RTK watchlistApi  
- [x] MWL-3 — merge/key pure helpers + tests  
- [x] MWL-4 — WatchlistProvider context  
- [x] MWL-5 — StarButton + Markets/Detail wiring  
- [x] MWL-6 — Watchlist tab + WatchlistPage + list organisms  
- [x] MWL-7 — Quote enrichment + poll/empty/error  
- [x] MWL-8 — Tests polish + docs/board/changelog  

## Acceptance

- Star/unstar from Markets row and Coin detail  
- Watchlist tab lists saved pairs with exchange  
- Row opens Coin detail for that pair  
- clientId persisted across reloads  
- Optimistic updates; max-200 error shown  
- Quotes when ticker available; poll pauses in background  
- No `modules/*/components`; RTK only under `libs/api`  
- Tests + docs closed out  

## Out of scope

- Auth / multi-device accounts  
- Price alerts  
- Product web (`frontend/`) watchlist UI  
- Backend durable store  
- Pump scanner / AI  

## MR grouping

`(MWL-1…3)` · `(MWL-4…5)` · `(MWL-6…7)` · `(MWL-8)`
