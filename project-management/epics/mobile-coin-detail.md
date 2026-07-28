# Epic E: Mobile coin detail + indicators

**Priority:** P1 (mobile track, after markets dashboard)  
**Status:** done  
**Depends on:** Epic D mobile markets (`feature/mobile-spot-markets` / MMKT)  
**Branch:** `feature/mobile-coin-detail` (from develop after markets merges, or stacked on markets)  
**Design plan:** `docs/design/mobile-coin-detail.md`  
**Feature:** `docs/features/mobile-coin-detail.md`  
**Web parity:** `docs/features/coin-detail.md` (DET-1…4 done)  
**Tasks folder:** `project-management/tasks/mobile/detail/`

## Goal

From the mobile markets list, open a **coin detail** screen for `exchange` + `symbol`: 24h ticker, supply, interval selector, OHLCV chart, RSI/EMA — using **Atomic Design** + **View + ViewModel**, RTK Query, and module context where needed. Primary runtime remains **Chrome** (`npm run web`).

## APIs (existing backend)

| UI | Path |
|----|------|
| Intervals | `GET /api/v1/market/intervals` |
| Ticker 24h | `GET /api/v1/market/ticker/24h` |
| Supply | `GET /api/v1/market/supply` |
| Candles | `GET /api/v1/market/candles` |
| Indicators | `GET /api/v1/market/indicators` |

Field-level mapping: reuse web **DET-A** / **DET-B** analysis under `tasks/frontend/detail/`.

## Tasks

- [x] MDET-1 … MDET-7 (see board + `tasks/mobile/detail/`)

## Acceptance

- Tap market row → detail screen with exchange/symbol params
- Header + stats + interval control work
- Chart + RSI/EMA render on RN web (chart host per design)
- Loading/error per section; poll pauses when unfocused
- All UI from `src/components/` Atomic layers (no module components folder)
- Tests for helpers + page ViewModels

## Out of scope

- Watchlist / alerts  
- Indicator batch API  
- Native-only chart packages that require Expo  
