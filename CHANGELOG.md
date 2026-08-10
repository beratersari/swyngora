# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Live spot order books:** Binance, Coinbase, and Bybit keep a local book over each venue’s depth websocket; a gap or drop invalidates and resyncs instead of serving stale data (`docs/features/order-book.md`)

### Added
- **Market-wide order book:** combined Binance + Coinbase + Bybit live depth in the same ±% price band with total buy/sell notional, pressure, and imbalance (`GET /api/v1/market/orderbook/combined`, MCP `analyze_market_orderbook`) (`docs/features/order-book.md`)
- **Order-book alerts:** create imbalance or wall alerts on the existing alert API; the background checker uses the live local book and does not re-fire while the same condition stays true (`POST /api/v1/alerts` `kind=imbalance|wall`, MCP `create_orderbook_alert`) (`docs/features/price-alerts.md`)
- **Spot order-book analysis:** buy/sell pressure, notional imbalance, and large walls from live depth within ±`rangePct` of mid (default 2%, not only the top rows) on Binance, Coinbase, and Bybit (`GET /api/v1/market/orderbook`, MCP `analyze_spot_orderbook`) (`docs/features/order-book.md`)
- **Spot order book:** grouped bid/ask depth with buy/sell wall flags and suggested price steps (`GET /api/v1/market/orderbook`, MCP `get_spot_orderbook`); grouping is done on the backend; coin detail shows the ladder next to the chart (`docs/features/order-book.md`)
- **Paper trade idempotency keys:** `Idempotency-Key` header or `idempotencyKey` on market/pending/margin place and margin close; same key + same request returns the original fill; different request with that key is 409 (`docs/features/paper-trading.md`)
- **Paper portfolio export/import:** backup and restore owned books (cash, positions, trades, open orders, tax lots, recurring buys, margin, shares) via the existing export/import jobs; merge skips existing id/name, replace wipes owned books first (`docs/features/user-data-export.md`, `docs/features/user-data-import.md`)
- **Paper trading fees and slippage:** per-exchange taker fee and adverse slippage on market, pending, recurring, and margin fills; buy cash and tax-lot cost include the fee; sell realized PnL is after the fee; pending buy reservations cover worst-case slip + fee (`GET /api/v1/portfolio/trading-costs`, MCP `get_paper_trading_costs`) (`docs/features/paper-trading.md`)
- **Paper tax lots:** each buy opens a lot (qty, price, time); sells choose FIFO or LIFO (`lotMethod`); partial lots keep remaining qty; market, pending, and recurring fills share the same ledger (`GET /api/v1/portfolio/lots`) (`docs/features/paper-trading.md`)
- **Realtime WebSocket:** `GET /api/v1/ws` subscribe/unsubscribe selected coin prices and one paper portfolio’s order/position/cash events; reconnect resubscribes with snapshots; access checked per book; frontend uses the stream instead of polling when connected (`docs/features/realtime.md`)
- **Paper cash transfer:** owner-only move of available cash between your own books (`POST /api/v1/portfolio/transfers`); both histories get `transfer_out`/`transfer_in` with counterpart book name (not a deposit/withdrawal); MCP `transfer_portfolio_cash`; Telegram `/transfer` (`docs/features/paper-trading.md`)
- **Paper portfolio sharing:** owner grants `viewer` (read snapshot/trades/performance) or `trader` (also place/cancel orders) to another clientId; deposit/withdraw/delete/share stay owner-only; `POST/GET/PATCH/DELETE /api/v1/portfolio/shares` and `GET /api/v1/portfolios/shared` (`docs/features/paper-trading.md`)
- **Multiple named paper portfolios:** each client can create up to 20 books with their own cash, positions, orders, and performance; `GET /api/v1/portfolios`, `PATCH`/`DELETE /api/v1/portfolios/{id}`, optional `portfolioId` / `X-Portfolio-Id` on actions; MCP `list_portfolios` / `rename_portfolio` / `delete_portfolio`; Telegram `/portfolio list` `/use` (`docs/features/paper-trading.md`)
- **Per-account API keys:** named `read` or `trade` keys (`POST/GET/DELETE /api/v1/account/api-keys`); bots use `swy_…` secrets instead of the process master token (`docs/features/api-keys.md`)
- **Paper cash in/out:** `POST /portfolio/deposits` and `/withdrawals` plus `GET /cash-movements` history; deposits do not count as P&L; MCP + Telegram `/deposit` `/withdraw` `/cash` (`docs/features/paper-trading.md`)
- **Telegram paper trading:** `/portfolio` view/create plus `/buy` and `/sell` with last-price preview and Confirm/Cancel inline buttons (`docs/features/telegram-bot.md`)
- **Paper portfolio performance:** `GET /api/v1/portfolio/performance?period=1d|1w|1m|3m` returns equity time series plus period P&L amount and percent; 15m snapshot worker; MCP `get_portfolio_performance` (`docs/features/paper-trading.md`)
- **Paper order amend:** `GET`/`PATCH /api/v1/portfolio/orders/{id}` to change trigger price and remaining size of an open GTC limit/stop without canceling; same id, reservation recalc, 409 on concurrent fill; MCP `get_portfolio_order` / `amend_portfolio_order` (`docs/features/paper-trading.md`)
- **Paper cancel-all:** `POST /api/v1/portfolio/orders/cancel-all` cancels every open paper order or one market (`symbol`); MCP `cancel_all_portfolio_orders`
- **Named recurring buys:** plan `name` plus `weekly`+`weekday`, `monthly`+`dayOfMonth` (salary day), and `interval`+`intervalHours` (e.g. every 12h); `PATCH` to rename/reschedule; MCP `update_recurring_buy`
- **Allocation baskets:** named target mixes (e.g. 50/30/20) with live drift; **user-triggered** rebalance only (`POST .../baskets/{id}/rebalance`); no auto-rebalance (`docs/features/allocation-baskets.md`)
- **Risk limits:** optional `maxDailyLossPct` and `maxAssetWeightPct`; block new spot buys and new margin opens only — never close positions; GET returns live status for the settings screen (`docs/features/risk-limits.md`)
- **Mobile home dashboard** (`mobile/`): Home tab shows live favorites strip, top movers, highest volume, pump teaser, and quick chips; pull-to-refresh and AppState poll pause (`docs/features/mobile-home-dashboard.md`)
- **Mobile AI chat** (`mobile/`): Ask tab multi-turn chat via `POST /api/v1/ai/chat`; OpenAPI `postAiChat`; RTK mutation; context from coin detail; 503 UX when AI offline (`docs/features/mobile-ai-chat.md`)
- **Mobile chart pump overlays** (`mobile/`): coin detail chart toggles for pump/dump markers and high–low margin price lines (same pump events as the list section)
- **Mobile chart history pan** (`mobile/`): coin detail candle chart loads older bars when scrolling left (`endTime` pages, time-range viewport so empty left fills on all intervals; indicators limit follows series length)
- **Mobile icons** (`mobile/`): Lucide (`lucide-react-native` + `react-native-svg`) with `atoms/icon` wrapper; tab bar, favorites star, filters, back, language
- **Mobile localization** (`mobile/`): flexible i18next setup (`libs/i18n`) with en/tr catalogs, feature namespaces, language switcher on Home, tab/page copy wired for core screens
- **Mobile batch indicators** (`mobile/`): Favorites and Markets lists enrich rows with latest RSI via `POST /api/v1/market/indicators/batch` (≤50 symbols/exchange, AppState pause, partial-failure safe); coin detail series unchanged (`docs/features/mobile-batch-indicators.md`)
- **Mobile pumps radar** (`mobile/`): Pumps tab scans top-volume pairs via `/pumps/scan`; coin detail shows pump/dump events via `/pumps`; filters for lookback/threshold/direction (`docs/features/mobile-pumps.md`)
- **Mobile watchlist** (`mobile/`): star/unstar on Markets + Coin detail; Watchlist tab with ticker quotes; device clientId + optimistic local/server merge (`docs/features/mobile-watchlist.md`)
- **Mobile coin detail** (`mobile/`): CoinDetailPage from markets row — ticker/supply stats, interval toolbar, Lightweight Charts OHLCV + EMA overlays, RSI pane; RTK detail endpoints; Atomic organisms (`docs/design/mobile-coin-detail.md`)
- **Mobile multi-exchange markets dashboard** (`mobile/`): Markets tab lists Binance/Coinbase/Bybit spot via RTK Query; filters, sort, pagination, AppState poll pause; View+ViewModel (`docs/design/mobile-markets-dashboard.md`)
- **Product mobile scaffold** (`mobile/`): React Native (no Expo) + **react-native-web** for Chrome (`npm run web` → http://localhost:5180); Atomic components; modules own pages with View+ViewModel; RTK Query + OpenAPI codegen; brand tokens shared with frontend


### Added
- **Frontend i18n:** i18next + react-i18next with `en`/`tr` catalogs (`libs/i18n/`), language switcher, Ant Design locale sync; UI copy moved off hard-coded English

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
