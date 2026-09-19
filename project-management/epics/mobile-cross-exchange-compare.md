# Epic L: Mobile cross-exchange coin comparison

**Priority:** P1 (mobile track — after category discovery)  
**Status:** done  
**Depends on:** Epic E coin detail (done)  
**Branch:** `feature/mobile-cross-exchange-compare` (from latest `develop`)  
**Design plan:** `docs/design/mobile-cross-exchange-compare.md`  
**Feature:** `docs/features/mobile-cross-exchange-compare.md`  
**Analysis:** `project-management/tasks/mobile/cross-exchange/MCROSS-A.md`  
**Tasks folder:** `project-management/tasks/mobile/cross-exchange/`

## Goal

On **coin detail**, show the same base asset’s **24h ticker metrics across Binance, Coinbase, and Bybit** by composing existing `GET /ticker/24h` calls with client-side symbol mapping. Atomic Design + View + ViewModel; primary runtime **Chrome**.

## APIs (existing backend — no new endpoints in v1)

| UI | Path |
|----|------|
| Per-venue quote | `GET /api/v1/market/ticker/24h?exchange=&symbol=` |
| Venue list (optional) | `GET /api/v1/market/exchanges` |

Symbol mapping: e.g. `BTCUSDT` → Coinbase `BTC-USD` (try candidates; backend normalizes Coinbase form).

## Tasks

- [x] MCROSS-A — Field matrix + symbol mapping analysis  
- [x] MCROSS-1 — Constants + cross-exchange helpers + tests  
- [x] MCROSS-2 — Atomic compare UI (row + list organism)  
- [x] MCROSS-3 — Coin detail ViewModel: parallel tickers + mapping  
- [x] MCROSS-4 — Wire section into CoinDetailPage + navigate to venue  
- [x] MCROSS-5 — Loading / partial failure / disclaimer / i18n  
- [x] MCROSS-6 — Tests polish  
- [x] MCROSS-7 — Docs / board / changelog closeout  

## Acceptance

- Detail shows multi-venue rows for major pairs when available  
- Source venue uses route symbol; others use mapped candidates  
- One venue failure does not blank the section  
- Tap other venue opens its detail route  
- Polling pauses when unfocused / backgrounded  
- No `modules/*/components`; RTK under `libs/api` only  
- Tests + docs closed  

## Out of scope

- New OpenAPI compare endpoint (optional follow-up)  
- FX conversion between USDT and USD  
- Alerts / paper trading / arb execution  
- Web frontend compare UI  

## MR grouping

`(A+1)` · `(2–3)` · `(4–5)` · `(6–7)`
