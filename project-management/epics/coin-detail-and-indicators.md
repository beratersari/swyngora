# Epic C: Coin detail + technical indicators (frontend)

**Priority:** P1  
**Status:** done (analysis DET-A/B + implementation DET-1…4)  
**Analysis doc:** `docs/features/coin-detail.md`  
**Depends on:** Epic B (Markets list) done  

## Goal

Give users a single-pair **detail** surface: 24h context, supply, OHLCV chart, and RSI/EMA analysis — linked from the Markets table. Backend APIs already exist; this epic is product UI + RTK wiring only.

## Scope (analysis outcome)

| In scope | Out of scope (later) |
|---|---|
| Coin detail route from Markets row | Watchlist star on detail (WL-1) |
| 24h ticker + supply stats | Paper trading / alerts |
| Lightweight Charts candles | Extra indicators (MACD, BB, …) — no API yet |
| RSI series pane + EMA on price | Cross-exchange compare page |
| Shared interval for chart + indicators | Auth / multi-user |

## APIs (existing backend)

| Use | Method | Path |
|---|---|---|
| Intervals | `GET` | `/api/v1/market/intervals?exchange=` |
| 24h ticker | `GET` | `/api/v1/market/ticker/24h?exchange=&symbol=` |
| Supply | `GET` | `/api/v1/market/supply?symbol=` or `?asset=` |
| Candles | `GET` | `/api/v1/market/candles?…` |
| Indicators | `GET` | `/api/v1/market/indicators?…` |

Optional later: `POST /api/v1/market/indicators/batch` (list columns, not detail).

## Tasks

### Analysis (do first — no code)

| ID | Task | Status | Detail level |
|---|---|---|---|
| DET-A | Coin detail page analysis | done | Full: every endpoint query/response field, mapping, orchestration, UI matrix → `tasks/frontend/detail/DET-A.md` |
| DET-B | Technical indicators on detail analysis | done | Full: GET indicators + batch (not used), RSI/EMA logic, bands, overlays → `tasks/frontend/detail/DET-B.md` |

### Implementation (after analysis accepted)

| ID | Task | Status |
|---|---|---|
| DET-1 | RTK detail endpoints (candles, ticker, supply, intervals, indicators) | done |
| DET-2 | Coin detail page shell + header/stats + route from Markets | done |
| DET-3 | Candle chart (Lightweight Charts) + interval/limit toolbar | done |
| DET-4 | RSI/EMA panel + EMA overlay + tests/docs | done |

## Acceptance

### Analysis phase
- [x] PM epic + tasks exist under `project-management/`
- [x] Page layout, states, and design decisions written
- [x] Endpoint mapping complete

### Implementation phase (DET-A + DET-B applied)
- [x] DET-1 RTK endpoints
- [x] DET-2 page shell + route + header/stats
- [x] DET-3 candles + toolbar
- [x] DET-4 RSI/EMA panel + overlays + unit tests

## Notes

Product UI: `frontend/src/components/pages/CoinDetailPage/`.  
Open: http://localhost:5174/markets/binance/BTCUSDT (or row click on Markets).
