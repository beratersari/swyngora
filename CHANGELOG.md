# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Frontend layout (Option A):** removed `src/features/`; domain UI lives under `components/organisms/`; pages under `components/pages/` only; decision `project-management/decisions/002-no-features-folder-atomic-only.md`
- **Frontend brand palette:** navy/indigo blues replaced with green system — Rich Black `#000F0F`, Dark Green `#032221`, Bangladesh Green `#03624C`, Mountain Meadow `#4FD4A5`, Caribbean Green `#00FF81`, Anti-Flash White `#F1F7F6`, secondary Pine–Mint, neutrals Stone/Pistachio (`frontend/src/styles/tokens/colors.ts`)

### Added
- **Multi-agent AI assistant** (`ai/`): LangGraph orchestrator with market, web, X/Twitter-signal, and analyst specialists; Ollama + Grok only
- **Go MCP** embedded in `cmd/server` at `/mcp` (in-process tools); optional stdio `cmd/mcp`
- **Telegram `/ask` and `/ai`** plus `POST /api/v1/ai/chat` proxy; optional AI auto-start child process
- AGENTS.md rule: new agent-useful features must ship with MCP tools in the same MR; restart backend after server changes
- Docs: `docs/features/ai-assistant.md`, ADR 0002 multi-agent + MCP
- **Product coin detail + indicators UI** (`frontend/`): route `/markets/:exchange/:symbol`, Lightweight Charts OHLCV + EMA overlays, RSI pane (30/70 bands), ticker/supply stats; RTK endpoints for candles, ticker, supply, intervals, indicators (`docs/features/coin-detail.md`)
- **Frontend design + scaffold:** system design, project-init epic design, multi-exchange spot markets feature doc, `frontend/` libs/Atomic skeleton, GitLab PM definitions (`docs/pm/`)
- **Frontend stack decisions:** **Ant Design** + **TradingView Lightweight Charts**; local **`project-management/`** board (INIT/MKT tasks)
- **Product frontend scaffold** (`frontend/`): Vite + React + TS, Ant Design dark theme, Lightweight Charts host, RTK Query `libs/api`, OpenAPI codegen, Markets placeholder; scripts `dev`/`build`/`test`/`codegen:api`
- **Frontend styling:** **styled-components** only; colocated `*.styles.ts` (no CSS modules)
- **Frontend design system:** brand palette `#111844` / `#4B5694` / `#7288AE` / `#EAE0CF`, type scale (DM Sans + JetBrains Mono), `Text` + `Skeleton` atoms, shared `isLoading` contract (`docs/design/frontend-design-system.md`)
- **Multi-exchange spot markets UI** (`frontend/`): exchange tabs, search/quote/tag filters, sortable Ant Table, URL query sync, 10s poll with visibility pause (`GET /exchanges`, `/tags`, `/spot`)
- **Telegram bot** integrated in backend (`transport/telegram`): market commands, watchlist, `/lowmcap` / `/lowmcap all` (no AI; enabled via `TELEGRAM_BOT_TOKEN`)
- Technical indicators: **RSI** (Wilder) and **EMA** via `GET /api/v1/market/indicators`
- OpenAPI for **`POST /api/v1/market/indicators/batch`**; document `exchange` on ticker/intervals/tags
- **Watchlist** API (`/api/v1/watchlist`) + dashboard stars / filter (client id + localStorage)
- Multi-exchange spot data: **Coinbase** + **Bybit** via `?exchange=`; `GET /api/v1/market/exchanges`
- Binance product-catalog **tags** on spot markets: `tags` field, `tag`/`tags` filter (OR), `sort=tags`, and `GET /api/v1/market/tags`
- Simple frontend tag filter dropdown and Tags column

### Fixed
- Watchlist `Add` max-items race (enforced under store lock); reject empty/`default` clientId
- Indicator series no longer collapses bad closes (fail instead of inventing RSI/EMA gaps)
- Coinbase symbol normalization shared with watchlist (`BTCUSD` → `BTC-USD`)
- Empty/unparseable volumes sort nulls-last (not as zero); tag sort uses sorted tag join
- Binance product-meta/exchangeInfo singleflight detaches caller context
- Bybit: only `Trading` instruments; candle `CloseTime` from interval; safer error mapping
- Coinbase: hard-fail invalid OHLC rows; error if product pagination hits safety cap
- Telegram: fail-closed without allowlist; `/rsi` accepts either arg order; escape HTML errors; free-text not `/price`
- JSON POST/PUT body size capped (1 MiB); CORS origin allowlist via `CORS_ALLOW_ORIGINS`
- Simple frontend: Coinbase dashboard uses `quote=USD`; watchlist DELETE tombstones prevent re-merge
- Market-cap ranking: nulls last, no infinite max without price, collapse multi-quote pairs, refuse empty-supply mcap sorts
- Supply snapshot: atomic replace, USDT-pair preference, strict bapi success checks, last-good retained on failure; **retry with backoff** on failed refresh; default **48h safety TTL**
- Non-crypto filter: fail-closed on empty/soft catalog **and** when catalog is down with no last-good snapshot (spot list errors instead of listing equities)
- Candle/ticker thundering herd (singleflight); unbounded candle cache keys (range queries uncached + max entries)
- Ingress per-IP rate limit with **max bucket map**; watchlist **Add max items** + **max clients**; indicator batch **process-wide** upstream semaphore
- Sanitized public API errors; zero-duration config footguns
- Simple frontend: detail-page load race (stale symbol paint); watchlist **merge** sync (no wipe after offline adds); multi-exchange star paint vs click; tiny prices no longer format as `0`; XSS formatters; supply `asOf` cues

### Security
- Telegram bot does not start with token alone unless `TELEGRAM_ALLOW_ALL=true` or chat allowlist is set
- Configurable CORS (default `*` for local dev; restrict in production)


### Changed

- Supply (circulating / total / max) comes **only from Binance** marketing symbol list; CoinGecko adapter removed
- Supply snapshot still daily @ 03:00 UTC (+ startup); request path remains cache-only
- Max supply is null when Binance does not publish a hard cap
- Spot list and supply exclude non-crypto products (`bStocks` e.g. NVDAB/TSLAB, `tCommodities` e.g. PAXG)

### Added

- Daily supply/mcap snapshot refresh (default 03:00 UTC); user requests are cache-only
- Binance spot market list with search, metric sort, and pagination (`GET /api/v1/market/spot`)

## [0.1.0] - 2026-07-25

### Added

- Go backend (N-layered) with OpenAPI contract for market data
- Binance candlesticks (`/api/v1/market/candles`) with multi-interval support
- Binance 24h ticker including base and quote volume (`/api/v1/market/ticker/24h`)
- Asset circulating / total / max supply via free CoinGecko (`/api/v1/market/supply`)
- In-memory TTL caches with background cleanup for candles, ticker, and supply
- `simple-frontend/` static test harness (product UI reserved under `frontend/`)
- Feature docs and ADR for data-source choice

[0.1.0]: https://nova.teachx.ai/trace-analysis/swyngora/-/tags/v0.1.0
