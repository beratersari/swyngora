# Frontend board

## Stack (locked)

- UI: **Ant Design**
- Charts: **TradingView Lightweight Charts**
- Data: RTK Query + OpenAPI in `libs/api`

## Epic A — Project initialization (P0)

| ID | Task | Status |
|---|---|---|
| INIT-1 | Scaffold Vite React TypeScript app | done |
| INIT-2 | Lint, format, Vitest, path aliases | done |
| INIT-3 | libs + Atomic folders + Ant Design provider | done |
| INIT-4 | libs/api store + RTK baseApi | done |
| INIT-5 | OpenAPI codegen → libs/api/generated | done |
| INIT-6 | App shell, router, env, Markets placeholder (Ant Layout) | done |
| INIT-7 | Docs: README + AGENTS + changelog | done |
| INIT-8 | Install/configure Ant Design + theme tokens | done |
| INIT-9 | Add lightweight-charts + thin chart wrapper atom/molecule | done |

## Epic B — Multi-exchange spot markets (P1, done)

| ID | Task | Status |
|---|---|---|
| MKT-1 | RTK market endpoints in libs/api | done |
| MKT-2 | Markets page shell + ExchangeTabs (Ant) | done |
| MKT-3 | MarketsTable (Ant Table) + formatters | done |
| MKT-4 | Toolbar filters (search, quote, tags, sort) | done |
| MKT-5 | Pagination + URL query sync | done |
| MKT-6 | Live poll + visibility pause | done |
| MKT-7 | Empty/error UX + tests | done |

## Epic C — Coin detail + technical indicators (P1)

**Epic:** `epics/coin-detail-and-indicators.md`  
**Analysis doc:** `docs/features/coin-detail.md`  
**Process:** analysis tasks first (DET-A/B); implementation (DET-1…4) only after acceptance.

### Analysis (no code) — full field-level specs

| ID | Task | Status |
|---|---|---|
| DET-A | Coin detail: route + intervals/ticker/supply/candles (all fields + logic) | done |
| DET-B | Indicators: GET /indicators (+ batch out-of-scope), RSI/EMA mapping | done |

See `tasks/DET-A.md` and `tasks/DET-B.md` for complete contracts.

### Implementation (DET-A + DET-B applied)

| ID | Task | Status |
|---|---|---|
| DET-1 | RTK detail endpoints (candles, ticker, supply, intervals, indicators) | done |
| DET-2 | Coin detail page shell + header/stats + route from Markets | done |
| DET-3 | Candle chart + interval/limit toolbar | done |
| DET-4 | RSI/EMA panel + EMA overlay + tests | done |

## Later (not started)

| ID | Task | Status |
|---|---|---|
| WL-1 | Watchlist UI | backlog |

## Status legend

`todo` · `in_progress` · `blocked` · `done` · `backlog`
