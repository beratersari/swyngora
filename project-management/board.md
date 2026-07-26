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

## Epic B — Multi-exchange spot markets (P1, ready — Epic A done)

| ID | Task | Status |
|---|---|---|
| MKT-1 | RTK market endpoints in libs/api | todo |
| MKT-2 | Markets page shell + ExchangeTabs (Ant) | todo |
| MKT-3 | MarketsTable (Ant Table) + formatters | todo |
| MKT-4 | Toolbar filters (search, quote, tags, sort) | todo |
| MKT-5 | Pagination + URL query sync | todo |
| MKT-6 | Live poll + visibility pause | todo |
| MKT-7 | Empty/error UX + tests | todo |

## Later (not started)

| ID | Task | Status |
|---|---|---|
| DET-1 | Coin detail page + Lightweight Charts candle view | backlog |
| DET-2 | RSI/EMA series on/near chart | backlog |
| WL-1 | Watchlist UI | backlog |

## Status legend

`todo` · `in_progress` · `blocked` · `done` · `backlog`
