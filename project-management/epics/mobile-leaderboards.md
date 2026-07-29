# Epic M: Mobile gainers / losers / high-volume leaderboards

**Priority:** P1 (mobile track — after cross-exchange compare)  
**Status:** done  
**Depends on:** Markets + Home (done)  
**Branch:** `feature/mobile-leaderboards` (from latest `develop`)  
**Design plan:** `docs/design/mobile-leaderboards.md`  
**Feature:** `docs/features/mobile-leaderboards.md`  
**Analysis:** `project-management/tasks/mobile/leaderboards/MLEAD-A.md`  
**Tasks folder:** `project-management/tasks/mobile/leaderboards/`

## Goal

Ship **first-class leaderboards** for 24h **gainers**, **losers**, and **high quote volume** using existing `GET /api/v1/market/spot` sorts — full lists (pagination), exchange/quote filters, and Home “See all” deep links. Atomic Design + View + ViewModel; primary runtime **Chrome**.

## APIs (existing backend — no new endpoints)

| Board | Path |
|-------|------|
| Gainers | `GET /spot?sort=priceChangePercent&order=desc` |
| Losers | `GET /spot?sort=priceChangePercent&order=asc` |
| Volume | `GET /spot?sort=quoteVolume&order=desc` |

Shared: `exchange`, `quote`, `status=TRADING`, `limit`/`offset`.

## Tasks

- [x] MLEAD-A — Spot sort field matrix + UX decisions  
- [x] MLEAD-1 — Constants + leaderboard query helpers + tests  
- [x] MLEAD-2 — Atomic segment / list chrome (reuse markets row/list)  
- [x] MLEAD-3 — LeaderboardsPage ViewModel  
- [x] MLEAD-4 — LeaderboardsPage View + navigation routes  
- [x] MLEAD-5 — Home “See all” deep links + Markets entry  
- [x] MLEAD-6 — Loading / empty / error / poll / i18n  
- [x] MLEAD-7 — Tests polish + docs / board / changelog  

## Acceptance

- User can open full gainers, losers, and volume boards  
- Exchange + quote chips change the ranking source  
- Infinite scroll or load-more works  
- Home movers/volume “See all” lands on the correct board  
- Polling pauses when unfocused / backgrounded  
- No `modules/*/components`; RTK under `libs/api`  
- Tests + docs closed  

## Out of scope

- New backend endpoints  
- Multi-timeframe (1h/7d) boards  
- Alerts on rank changes  
- Web frontend leaderboard redesign  

## MR grouping

`(A+1)` · `(2–3)` · `(4–5)` · `(6–7)`
