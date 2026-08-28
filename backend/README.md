# Swyngora Backend

Go HTTP API for market data across **Binance**, **Coinbase**, **Bybit** (crypto spot), **Nasdaq**, and **BIST** (cash equities). Crypto circulating supply remains from the Binance daily snapshot and is **not** applied to stocks. Equity market caps come from the Nasdaq.com screener (USD) and the public TradingView Turkey scanner (TRY).

## Architecture (N-layered)

| Layer | Path | Role |
|---|---|---|
| Transport | `internal/transport/http` | HTTP + WebSocket handlers, CORS, rate limit, JSON mapping |
| Transport | `internal/transport/telegram` | Optional Telegram bot (long-poll → same services) |
| Application | `internal/service/market` | Validation + use-case orchestration |
| Application | `internal/service/realtime` | WebSocket hub: price pump + portfolio event fan-out |
| Domain | `internal/domain` | Entities, ports, sentinel errors |
| Infrastructure | `internal/adapter/*` | Binance (market + supply), Coinbase, Bybit, **equities (Nasdaq screener + BIST TradingView scanner; Yahoo for candles)**, TTL cache, **SQLite watchlist** |
| Platform | `internal/platform/config` | Env config |

OpenAPI contract: [`api/openapi/openapi.yaml`](api/openapi/openapi.yaml).

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/api/v1/realtime` | WebSocket protocol description |
| `GET` | `/api/v1/ws` | WebSocket: live prices + paper portfolio events |
| `GET` | `/api/v1/market/exchanges` | Supported venues |
| `GET` | `/api/v1/market/fx` | Spot FX rates (USD base) for display conversion |
| `GET` | `/api/v1/market/intervals` | Candle intervals (per `exchange`) |
| `GET` | `/api/v1/market/tags` | Unique Binance product-catalog tags (crypto) |
| `GET` | `/api/v1/market/spot?q=btc&quote=USDT&tag=Meme&sort=quoteVolume` | List/search/filter/sort spot markets |
| `GET` | `/api/v1/market/indicators?symbol=BTCUSDT&interval=1h` | RSI (Wilder) + EMA series |
| `POST` | `/api/v1/market/indicators/batch` | Latest RSI/EMA for up to 50 symbols (bounded concurrency) |
| `GET` | `/api/v1/market/delist-schedule` | Cached spot delist schedule (`exchange=`; halt `delistTime` + `announcedAt`; last 30 days + next ~31 days; Binance needs `BINANCE_API_KEY`, Bybit is public announcements) |
| `GET` | `/api/v1/market/post-delist` | Off-venue last + candles after this exchange halted the pair (other listed venue or CoinGecko USD; informational) |
| `GET` | `/api/v1/market/orderbook` | Grouped live spot book + ±% pressure/wall analysis |
| `GET` | `/api/v1/market/orderbook/history` | Stored book at a time, or a newest-first list |
| `GET` | `/api/v1/market/orderbook/history/compare` | Which price levels gained or lost liquidity between two times |
| `GET` | `/api/v1/market/orderbook/icebergs` | Same-price clip eaten then refilled (bid and ask) |
| `GET` | `/api/v1/market/orderbook/combined` | Market-wide pressure from all three venues in one price band |
| `GET` | `/api/v1/market/orderbook/impact` | Simulated market-order fill: average price, slippage, exhausted |
| `GET` | `/api/v1/market/orderbook/liquidity` | 0–100 liquidity score from ±0.1/0.5/1% depth; per venue + market-wide |
| `GET` | `/api/v1/market/orderbook/heatmap` | Resting bid/ask size over time (pre-warmed for all live crypto pairs; `window` seconds) |
| `GET` | `/api/v1/market/liquidations` | Rolling 5m/1h/4h/24h futures long/short liquidations (Binance USD-M + Bybit linear) |
| `GET` | `/api/v1/market/holders` | Crypto holder count, concentration, and top wallets (CMC → Coin Metrics → GeckoTerminal → Ethplorer → Routescan → Tronscan; CryptoID fallback for UTXO coins like PIVX). Known addresses include a public `label` |
| `GET` | `/api/v1/market/open-interest` | Current futures OI + 5m/1h/4h/24h change (Binance USD-M + Bybit linear); includes funding |
| `GET` | `/api/v1/market/funding-rate` | Predicted next perpetual funding + recent settlements (Binance USD-M + Bybit linear) |
| `GET` | `/api/v1/market/funding-arb` | Long cheaper-funding venue / short richer one; sized after-fee payout + spot-perp |
| `GET` | `/api/v1/market/funding-arb/scan` | After-fee winners only (published settlements in the hold window) |
| `GET` | `/api/v1/market/funding-arb/history` | Past after-fee stretches for one coin (`from`/`to`; first clock is entry only) |
| `POST`/`GET` | `/api/v1/funding-arb/watches` | Follow a pair; notify when after-fee net ≥ `minProfit` |
| `GET`/`DELETE` | `/api/v1/funding-arb/watches/{id}` | Get / delete a funding-arb follow |
| `GET` | `/api/v1/funding-arb/signals` | Open/closed min-profit crossings |
| `GET` | `/api/v1/market/long-short-ratio` | Account long/short ratio + recent 5m history (Binance USD-M + Bybit linear) |
| `GET` | `/api/v1/market/futures-history` | Durable stored OI / funding / long-short / liquidation history |
| `GET` | `/api/v1/market/liquidation-hunt` | Hypothetical per-venue hunt: spot size to reach liq zones + rough desk result |
| `GET` | `/api/v1/market/squeeze-risk` | Long/short squeeze risk scores per venue + OI-weighted combined |
| `GET` | `/api/v1/market/positioning` | Price+OI regime (buildup / unwinding / covering) per venue + combined market |
| `GET` | `/api/v1/market/venue-divergence` | Binance vs Bybit: same or opposite, which signals differ and why |
| `GET` | `/api/v1/market/taker-flow` | Aggressive futures buy vs sell volume (5m/1h/4h) per venue + combined |
| `GET` | `/api/v1/market/cvd` | Spot and futures CVD versus price; 15m/1h/4h/24h change; venue split and spot-vs-futures split |
| `GET` | `/api/v1/market/volume-profile` | Volume by price (POC + 70% value area, buy/sell); Binance and Bybit separately plus combined |
| `GET` | `/api/v1/market/around` | What happened around a time (before / during / after): price, volume, VWAP, vs typical, POC, sweeps, stored book/futures |
| `GET` | `/api/v1/market/around/compare` | How two times / moves of the same coin differed (price, volume, book, OI, sweeps) |
| `GET` | `/api/v1/market/around/moves` | Strongest recent up/down legs plus what happened during each |
| `GET` | `/api/v1/market/around/precursors` | What often changed before those moves, including conditions that fire together |
| `GET` | `/api/v1/market/around/similar` | Past important-move setups like the current tape; unique events; caller-picked after horizons |
| `GET` | `/api/v1/market/vwap` | Volume-weighted average price from a start time; last vs VWAP; Binance, Bybit, combined |
| `GET` | `/api/v1/market/absorption` | Large market buys/sells vs price hold; which side is absorbing and how strong |
| `GET` | `/api/v1/market/liquidity-sweeps` | Poke through a prior high/low that comes back; level, excursion, time, volume |
| `GET` | `/api/v1/market/volume-surge` | Current 5m/15m/1h volume vs that coin's typical; buy/sell split |
| `GET` | `/api/v1/market/volume-surge/scan` | Rank coins whose volume is much higher than their own typical |
| `GET` | `/api/v1/market/basis` | Perp vs spot/index premium or discount, trend, funding/OI read |
| `GET` | `/api/v1/market/correlation` | How similarly a coin moves with BTC and ETH (1h / 4h / 24h) |
| `GET` | `/api/v1/market/breadth` | How many followed coins are up vs down (1h / 4h / 24h) |
| `GET` | `/api/v1/market/volatility` | How much a coin moved (range + vs normal / BTC / ETH) |
| `GET` | `/api/v1/market/snapshot` | Price, volume, mcap, OI, funding, LS, and taker flow together |
| `GET` | `/api/v1/market/levels` | Support and resistance areas plus breakout strength |
| `GET` | `/api/v1/market/whales` | Large trades and liquidations, biggest first |
| `GET` | `/api/v1/market/pumps` | Mechanical pump/dump events for one symbol |
| `GET` | `/api/v1/market/pumps/scan` | Ranked pump hits across top-volume symbols |
| `GET` | `/api/v1/watchlist` | Get watchlist + `version` (`clientId` / optional `ownerClientId`) |
| `POST` | `/api/v1/watchlist/items` | Add symbol (owner or editor; optional `baseVersion`) |
| `DELETE` | `/api/v1/watchlist/items?exchange=&symbol=` | Remove symbol (`baseVersion` / If-Match) |
| `PUT` | `/api/v1/watchlist` | Replace list (owner; `baseVersion` + optional `baseItems` for multi-device) |
| `GET`/`POST`/`PATCH`/`DELETE` | `/api/v1/watchlist/shares` | List / grant / update role / revoke shares |
| `GET` | `/api/v1/watchlist/shared` | Lists shared with the caller |
| `GET` | `/api/v1/watchlist/audit` | Change history (who/when) |
| `GET` | `/api/v1/alerts` | List price alerts (`clientId` / `X-Client-Id`) |
| `POST` | `/api/v1/alerts` | Create price, imbalance, or wall alert |
| `GET` | `/api/v1/alerts/{id}` | Get one alert |
| `DELETE` | `/api/v1/alerts/{id}` | Delete an alert |
| `GET`/`PUT`/`DELETE` | `/api/v1/alerts/webhook` | Get / set / clear alert webhook URL |
| `POST` | `/api/v1/portfolio` | Create named paper book (starting balance, optional name) |
| `GET` | `/api/v1/portfolio` | Cash, positions, P&L for selected book (`portfolioId`) |
| `GET` | `/api/v1/portfolios` | List paper books |
| `GET` | `/api/v1/portfolios/shared` | Books shared with the caller |
| `PATCH`/`DELETE` | `/api/v1/portfolios/{id}` | Rename or delete a book |
| `POST`/`GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/shares` | Share a book (`viewer`/`trader`), list, update, revoke |
| `POST` | `/api/v1/portfolio/deposits` | Add virtual cash |
| `POST` | `/api/v1/portfolio/withdrawals` | Withdraw available virtual cash |
| `POST` | `/api/v1/portfolio/transfers` | Move cash between own books (owner only) |
| `GET` | `/api/v1/portfolio/cash-movements` | Deposit / withdraw / transfer history |
| `GET` | `/api/v1/portfolio/performance` | Equity history + period P&L (`1d`/`1w`/`1m`/`3m`) |
| `GET`/`PUT`/`DELETE` | `/api/v1/portfolio/risk-limits` | Optional risk brakes (daily loss %, max coin weight); block new buys/margin only |
| `GET` | `/api/v1/portfolio/trading-costs` | Per-exchange paper taker fee and slippage |
| `POST` | `/api/v1/portfolio/orders` | Paper market or pending (`limit_buy` / `limit_sell` / `stop_loss`); optional `Idempotency-Key` |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (default: open) |
| `GET` | `/api/v1/portfolio/orders/{id}` | One pending order + last price + amend hints |
| `PATCH` | `/api/v1/portfolio/orders/{id}` | Amend open GTC limit/stop price and/or remaining size |
| `POST` | `/api/v1/portfolio/orders/cancel-all` | Cancel all open paper orders, or one market (`symbol`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel an open pending order |
| `GET` | `/api/v1/portfolio/trades` | Paper trade history |
| `GET` | `/api/v1/portfolio/lots` | Tax lots (FIFO/LIFO remaining buys) |
| `POST`/`GET` | `/api/v1/portfolio/recurring-buys` | Create / list named paper recurring buy (DCA) plans |
| `GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Get / update name-schedule / delete plan |
| `POST`/`GET` | `/api/v1/portfolio/baskets` | Create / list named allocation baskets (target % mix) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/portfolio/baskets/{id}` | Get with live drift / update / delete |
| `GET`/`POST` | `/api/v1/portfolio/baskets/{id}/preview` · `/rebalance` | Preview or user-triggered rebalance |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Recurring buy execution history |
| `POST`/`GET` | `/api/v1/portfolio/margin/orders` | Paper margin open (market/limit long/short 1x–10x; optional `Idempotency-Key`) / list |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel margin limit order |
| `GET` | `/api/v1/portfolio/margin/positions` | Open margin positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | Get margin position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full or partial margin close (optional `Idempotency-Key`) |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Stop-loss / take-profit |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |
| `POST`/`GET` | `/api/v1/price-diff/watches` | Create / list cross-exchange price difference watches |
| `GET`/`DELETE` | `/api/v1/price-diff/watches/{id}` | Get / delete watch |
| `GET` | `/api/v1/price-diff/opportunities` | List opportunities (`status=open\|closed\|all`) |
| `GET` | `/api/v1/price-diff/opportunities/{id}` | Get opportunity |
| `GET` | `/api/v1/price-diff/opportunities/{id}/quote` | Walk both books for a size (avg buy/sell, slippage, profit after fees, usable money, max size) |
| `GET` | `/api/v1/price-diff/quote` | Same walk without a stored opportunity (`buyExchange` / `sellExchange` / `notional`) |
| `GET` | `/api/v1/price-diff/quote/scan` | Rank every venue pair at one size (fees + live depth + how much money can be used) |
| `GET` | `/api/v1/price-diff/watches/{id}/quote` | Same scan using a watch's symbol and fees |
| `POST` | `/api/v1/scanner/rules` | Create RSI / MA crossover / volume scanner rule |
| `GET` | `/api/v1/scanner/rules` | List scanner rules |
| `GET` | `/api/v1/scanner/rules/{id}` | Get scanner rule |
| `DELETE` | `/api/v1/scanner/rules/{id}` | Delete scanner rule |
| `GET` | `/api/v1/scanner/results` | Scanner match history |
| `POST` | `/api/v1/scanner/backtests` | Start historical rule backtest |
| `GET` | `/api/v1/scanner/backtests` | List backtests |
| `GET` | `/api/v1/scanner/backtests/{id}` | Backtest progress/summary |
| `POST` | `/api/v1/scanner/backtests/{id}/cancel` | Cancel backtest |
| `GET` | `/api/v1/scanner/backtests/{id}/signals` | Backtest signals + 1/5/20d returns |
| `POST` | `/api/v1/export` | Start JSON/CSV export of own watchlist/shares/alerts/backtests/portfolios |
| `GET` | `/api/v1/export` | List export jobs |
| `GET` | `/api/v1/export/{id}` | Export progress / status |
| `POST` | `/api/v1/export/{id}/cancel` | Cancel pending/running export |
| `GET` | `/api/v1/export/{id}/download` | Download completed file (owner only; TTL) |
| `POST` | `/api/v1/import/preview` | Upload export file; get valid/invalid/willAdd counts |
| `POST` | `/api/v1/import/{id}/confirm` | Apply preview (`merge` or `replace`) in background |
| `GET` | `/api/v1/import` | List import jobs |
| `GET` | `/api/v1/import/{id}` | Import status / progress |
| `POST` | `/api/v1/import/{id}/cancel` | Cancel preview or running import |
| `GET` | `/api/v1/account` | Account status (active/closed, purgeAt, canReopen) |
| `POST` | `/api/v1/account/close` | Close account (7-day grace; data retained) |
| `POST` | `/api/v1/account/reopen` | Reopen within grace period |
| `GET`/`POST` | `/api/v1/account/api-keys` | List / create named API keys (`read` or `trade`) |
| `DELETE` | `/api/v1/account/api-keys/{id}` | Revoke a key |
| `POST` | `/api/v1/ai/chat` | Proxy to Python multi-agent assistant |
| `POST` | `/api/v1/ai/chat/stream` | NDJSON thinking/tool/final events (web Process panel) |
| `GET` | `/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=100` | OHLCV from Binance |
| `GET` | `/api/v1/market/ticker/24h?symbol=BTCUSDT` | 24h stats + base/quote volume |
| `GET` | `/api/v1/market/supply?asset=BTC` | Circulating supply (Binance product catalog) |

Intervals: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`.

Optional candle params: `startTime`, `endTime` (RFC3339 or Unix ms).

**Supply / mcap note:** Circulating / total / max supply is loaded from Binance’s public **marketing symbol list** on a **daily schedule** (default **03:00 UTC**, plus once on startup). Failed refreshes **retry with backoff** (1m→1h) before waiting for the next daily slot. User requests (`/supply`, spot mcap columns) **read from cache only**. Snapshots are **atomically replaced** on successful refresh (last-good retained on failure until safety TTL). Default **`SUPPLY_CACHE_TTL=48h`** so entries cannot live forever after a long outage (set `0` to never expire). Max is null when Binance does not define a hard cap (max mcap may show as infinite **only when a USD price exists**). Sorting by market-cap fields **collapses to one preferred quote pair per base** (USDT-first). Empty supply snapshot → market-cap sort returns `502`. **Delist rows** missing that snapshot get circulating supply from CoinGecko public `/coins/markets` so the Markets mcap column is not blank.

**Watchlist persistence:** client watchlists are stored in **SQLite** (default `data/watchlist.db`) so they survive process restarts. Configure path via `WATCHLIST_DB_PATH`. **Sharing:** owners grant `viewer` or `editor` access to other `clientId`s; editors may add/remove symbols only; all mutations write an audit log (`docs/features/watchlist-sharing.md`). **Multi-device sync:** each list has a monotonic `version`; send `baseVersion` on writes — non-overlapping adds auto-merge; delete-vs-update conflicts return **409** with both sides (`docs/features/watchlist-sync.md`).

**Price alerts:** above/below thresholds (`POST /api/v1/alerts`) with `mode=one_time` or `mode=repeating`. Optional webhook (`/api/v1/alerts/webhook`) supports `deliveryMode=immediate` or `hourly_digest`, plus **quiet hours** (`timeZone` + local start/end; midnight-crossing ranges OK). Delivery waits until quiet hours end; pending rows survive restarts. Webhook URLs are **SSRF-hardened** (no loopback/RFC1918/link-local/metadata; no HTTP redirects). Set `WEBHOOK_ALLOW_PRIVATE=true` only for local tests.

**Cross-exchange price diff:** watches (`/api/v1/price-diff/watches`) compare last prices on Binance, Coinbase, and Bybit after fees; opportunities record buy/sell venues when net edge exceeds `minNetDiffPct`. Open state is durable; no duplicate while open; re-opens after the edge drops and returns. Stale/missing prices skip that venue. Interval `PRICE_DIFF_CHECK_INTERVAL` (default `30s`). **Executable quote** (`/price-diff/quote` or `/opportunities/{id}/quote`) walks the buy asks and sell bids for a `notional` or `quantity` and returns average prices, slippage, profit after fees, usable money, and max still-profitable size. **Scan** (`/price-diff/quote/scan` or `/watches/{id}/quote`) ranks every venue pair at that size.

**Funding-arb follows:** `POST /api/v1/funding-arb/watches` stores a pair + `minProfit`. A background checker (`FUNDING_ARB_CHECK_INTERVAL`, default `30s`) re-quotes Binance vs Bybit and notifies the client's alert webhook (`type=funding_arb.triggered`) when after-fee horizon net is at least that floor and `trade` is present. Signals close and the watch re-arms when net falls below. SQLite path `FUNDING_ARB_DB_PATH` (default `data/fundingarb.db`). See `docs/features/funding-arb.md`.

**Paper trading:** virtual portfolio (`/api/v1/portfolio`) with starting cash, market buy/sell at last price **plus per-exchange slippage and taker fee**, pending limit/stop orders with cash/position **reservations** (buy reserve covers slip + fee), **partial fills**, **in-place amend** of open GTC limit/stop (`PATCH .../orders/{id}`), and **GTC/IOC/FOK** (+ optional GTC `expiresAt`) via the background filler, open positions, realized/unrealized P&L, trade history, **recurring buy (DCA) plans**, and **isolated margin** long/short (1x–10x, market/limit, liquidation, partial close, SL/TP). Simulated only — not real money. SQLite path `PORTFOLIO_DB_PATH` (default `data/portfolio.db`); order check interval `PORTFOLIO_ORDER_CHECK_INTERVAL` (default `15s`); recurring buy interval `RECURRING_BUY_INTERVAL` (default `30s`). Live prices + order/position events: `GET /api/v1/ws` (`docs/features/realtime.md`). Rates: `GET /api/v1/portfolio/trading-costs`.

**Indicator scanner:** create RSI / EMA crossover / volume-increase rules for the client's watchlist (`/api/v1/scanner/rules`). A background job evaluates rules on `SCANNER_CHECK_INTERVAL` (default `60s`), writes matches to history (`/api/v1/scanner/results`), and skips duplicates for the same rule + symbol + candle (`marketDataKey`). **Historical backtests** (`/api/v1/scanner/backtests`) re-run a rule over a date range for one symbol, track progress, support cancel, and report 1/5/20-day forward returns per signal. SQLite path `SCANNER_DB_PATH` (default `data/scanner.db`).

**Swing engine:** `GET /api/v1/market/swing?symbol=` analyzes one pair on **closed** 4h+1d bars (Wilder RSI/ADX/ATR, SuperTrend, MACD, volume/BB, BTC regime, min R:R 1.8 for trigger). `GET /api/v1/swing/setups` scans the watchlist (max 25). Informational only.

**User data export:** `POST /api/v1/export` queues a JSON or CSV dump of the caller's watchlist, shares, alerts, backtests, and paper portfolios. One active job per client; poll progress; cancel supported; download is owner-only; files expire (`EXPORT_FILE_TTL`, default 1h). See `docs/features/user-data-export.md`.

**User data import:** `POST /api/v1/import/preview` uploads a prior export and returns valid/invalid/willAdd counts; `confirm` with `merge` or `replace` applies in the background with progress/cancel and dedupe (portfolios: merge skips existing id/name; replace recreates owned books; extra non-UUID or globally colliding book ids are reminted so they cannot occupy another tenant’s first book). See `docs/features/user-data-import.md`.

**Account close:** `POST /api/v1/account/close` closes a `clientId` for 7 days (reopen allowed); product APIs and shared-list access stop; paper recurring plans are paused and open paper/margin orders canceled; workers skip the tenant. `X-Client-Id`, `?clientId=`, and JSON `clientId` must agree (else 400). Paper `PlaceOrder` and Telegram Confirm also refuse a closed owner. After grace, watchlists/shares/alerts/backtests/import-export files, paper books, price-diff watches, and funding-arb watches are purged. See `docs/features/account-close.md`.

**Hardening:** per-IP rate limits with **capped bucket map**; sanitized public errors; candle/ticker singleflight; bounded candle + watchlist client maps; non-crypto product filter **fails closed** without last-good catalog (no equities/commodities as crypto); indicator batch uses process-wide upstream semaphore; **webhook SSRF blocks** private destinations; paper portfolio mutations **serialized per `clientId`** (service mutex + store write lock); optional **`API_AUTH_TOKEN`** protects tenant APIs + `/mcp` (market GETs stay public); closed `clientId`s are blocked on tenant REST and on MCP tools that send `clientId`; **`MCP_ENABLED=false`** unmounts MCP.

**Auth note:** `clientId` / `X-Client-Id` is still a client-supplied label (not end-user login). For any network exposure set `API_AUTH_TOKEN` (and prefer non-`*` CORS). Users can mint named `swy_…` keys (`read` or `trade`) for bots so they do not share the master token (`docs/features/api-keys.md`). **User keys force their bound `clientId`** on REST body/query/form fields and MCP tool args (mismatches → 403); key-admin MCP tools are denied for user keys. Empty `API_AUTH_TOKEN` open mode is only allowed on **loopback** unless `ALLOW_OPEN_AUTH=true`. Default listen is `127.0.0.1:8080`. Full multi-user identity (JWT/session) is a separate follow-up.

## Run

```bash
# from backend/
go run ./cmd/server
# listens on 127.0.0.1:8080 by default
# optional Telegram bot starts when TELEGRAM_BOT_TOKEN is set
```

Optional local secrets (gitignored):

```bash
cp .env.example .env   # or place .env at repo root
# set TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID
go run ./cmd/server
```

After editing `.env` / bot tokens, **restart the server** so config reloads.

### Telegram bot commands

Enabled when `TELEGRAM_BOT_TOKEN` is non-empty. Calls **market**, **watchlist**, and **paper portfolio** services in-process (no HTTP hop).

| Command | Description |
|---------|-------------|
| `/price <symbol> [exchange]` | 24h ticker |
| `/spot [exchange] [query]` | Top by quote volume |
| `/lowmcap [exchange\|all] [n]` | Lowest circulating market cap |
| `/mcap <asset\|pair>` | Supply snapshot |
| `/rsi <symbol> [interval] [exchange]` | RSI + EMA |
| `/oi <symbol> [binance\|bybit\|all]` | Futures open interest + change + funding |
| `/funding <symbol> [binance\|bybit\|all]` | Current funding rate + recent history |
| `/fundingarb <symbol> [notional] [hours]` | Long/short venues + after-fee funding (`/fundingarb scan`) |
| `/ls <symbol> [binance\|bybit\|all]` | Long/short account ratio + recent history |
| `/exchanges` | Venues |
| `/watch` · `add` · `del` · `top` | Per-user watchlist (`tg-<user_id>`) |
| `/portfolio` · `/portfolio create [balance]` | Paper portfolio (`tg-<user_id>`) |
| `/deposit` · `/withdraw` `<amount> [note]` | Add or remove virtual cash |
| `/cash` | Deposit/withdraw history |
| `/buy` · `/sell` `<symbol> <qty> [exchange]` | Paper trade with Confirm / Cancel buttons |

See [`docs/features/telegram-bot.md`](../docs/features/telegram-bot.md).

### Environment

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `127.0.0.1:8080` | Listen address (loopback default) |
| `API_AUTH_TOKEN` | _(empty = open local mode on loopback only)_ | Master token for tenant routes + `/mcp` + key management; user `swy_…` keys also accepted |
| `ALLOW_OPEN_AUTH` | `false` | Permit empty `API_AUTH_TOKEN` when `HTTP_ADDR` is non-loopback (`0.0.0.0`, `:8080`, LAN) |
| `MCP_ENABLED` | `true` | Mount streamable MCP at `/mcp`; set `false` to disable |
| `WEBHOOK_ALLOW_PRIVATE` | `false` | Allow loopback/private webhook targets (local tests only; SSRF risk if true) |
| `TELEGRAM_BOT_TOKEN` | _(empty = disabled)_ | BotFather token; enables Telegram transport |
| `TELEGRAM_CHAT_ID` | — | Allowed chat id (or use `TELEGRAM_ALLOWED_CHAT_IDS`) |
| `TELEGRAM_ALLOWED_CHAT_IDS` | — | Comma-separated allowed chat ids |
| `TELEGRAM_ALLOW_ALL` | `false` | If true and allowlist empty, bot is public; otherwise token without allowlist **does not start** |
| `CORS_ALLOW_ORIGINS` | `*` | Comma-separated browser origins (`*` = any; set exact origins in production) |
| `BOT_DEFAULT_EXCHANGE` | `binance` | Default venue for bot commands |
| `BOT_POLL_TIMEOUT` | `30s` | Telegram long-poll wait |
| `BOT_LOWMCAP_LIMIT` | `10` | Default `/lowmcap` size (max 25) |
| `BINANCE_BASE_URL` | `https://api.binance.com` | Binance Spot REST base |
| `BINANCE_API_KEY` | _(empty)_ | Enables hourly spot delist schedule refresh |
| `DELIST_REFRESH_EVERY` | `1h` | Delist schedule poll interval |
| `DELIST_REFRESH_ON_STARTUP` | `true` | Fetch delist schedule once on start |
| `BINANCE_PRODUCT_BASE_URL` | `https://www.binance.com` | Host for marketing symbol list (supply) |
| `CMC_BASE_URL` | `https://api.coinmarketcap.com` | Public data-api host for holder snapshots |
| `HOLDERS_CACHE_TTL` | `1h` | Per-asset holder snapshot TTL; unpublished assets are negative-cached for the same window |
| `COINBASE_BASE_URL` | `https://api.coinbase.com` | Coinbase public market products |
| `COINBASE_EXCHANGE_URL` | `https://api.exchange.coinbase.com` | Coinbase public candles |
| `BYBIT_BASE_URL` | `https://api.bybit.com` | Bybit v5 public market API |
| `HTTP_CLIENT_TIMEOUT` | `15s` | Upstream HTTP timeout |
| `CANDLE_CACHE_TTL` | `30s` | Candle response TTL (latest-N queries only; ranges not cached) |
| `CANDLE_CACHE_MAX_ENTRIES` | `512` | Max candle cache keys (memory bound) |
| `TICKER_CACHE_TTL` | `15s` | Ticker response TTL |
| `ORDERBOOK_CACHE_TTL` | `2s` | Reserved TTL for any leftover REST book cache (live books are not cached) |
| `BINANCE_WS_URL` | `wss://stream.binance.com:9443` | Binance spot stream host for the live local book |
| `BINANCE_FUTURES_BASE_URL` | `https://fapi.binance.com` | Binance USD-M REST (open interest) |
| `BINANCE_FUTURES_WS_URL` | `wss://fstream.binance.com` | Binance USD-M stream for liquidations |
| `OPEN_INTEREST_CACHE_TTL` | `30s` | Futures open-interest snapshot TTL |
| `COINBASE_WS_URL` | `wss://ws-feed.exchange.coinbase.com` | Coinbase Exchange feed for the live local book |
| `BYBIT_WS_URL` | `wss://stream.bybit.com/v5/public/spot` | Bybit spot stream for the live local book |
| `BYBIT_LINEAR_WS_URL` | `wss://stream.bybit.com/v5/public/linear` | Bybit linear stream for liquidations |
| `ORDERBOOK_IDLE_TTL` | `90s` | Drop unused live depth streams |
| `ORDERBOOK_SYNC_TIMEOUT` | `8s` | Max wait for a synced local book |
| `REALTIME_PRICE_INTERVAL` | `5s` | How often subscribed WebSocket prices are pushed |
| `SUPPLY_CACHE_TTL` | `48h` | Safety TTL for supply snapshot (`0` = never expire until successful replace) |
| `SPOT_MARKET_CACHE_TTL` | `5s` | Joined spot prices TTL (exchangeInfo / product tags cached longer in-adapter) |
| `SUPPLY_REFRESH_HOUR` | `3` | Daily product-catalog snapshot hour (local to TZ) |
| `SUPPLY_REFRESH_MINUTE` | `0` | Daily snapshot minute |
| `SUPPLY_REFRESH_TZ` | `UTC` | Timezone for daily schedule (DST-safe calendar math) |
| `SUPPLY_REFRESH_ON_STARTUP` | `true` | Run one snapshot load on process start |
| `CACHE_CLEANUP_EVERY` | `1m` | Background expired-entry cleanup (must be > 0) |
| `RATE_LIMIT_RPS` | `40` | Per-IP request rate (0 disables) |
| `RATE_LIMIT_BURST` | `80` | Per-IP burst size |
| `WATCHLIST_DB_PATH` | `data/watchlist.db` | SQLite file for durable watchlists (created if missing) |
| `ALERTS_DB_PATH` | `data/alerts.db` | SQLite file for durable price alerts |
| `ALERT_CHECK_INTERVAL` | `30s` | How often active alerts are evaluated against last price |
| `WEBHOOK_DELIVERY_INTERVAL` | `5s` | Outbox drain interval for alert webhooks |
| `WEBHOOK_HTTP_TIMEOUT` | `10s` | Per-webhook HTTP timeout |
| `WEBHOOK_MAX_ATTEMPTS` | `8` | Permanent failure after this many delivery attempts |
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` | SQLite file for paper-trading portfolios |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` | How often open pending paper orders are evaluated |
| `RECURRING_BUY_INTERVAL` | `30s` | How often due recurring buy plans are evaluated |
| `PORTFOLIO_SNAPSHOT_INTERVAL` | `15m` | Equity sample cadence for performance charts |
| `PORTFOLIO_SNAPSHOT_RETENTION` | `2400h` | How long equity samples are kept (~100 days) |
| `MARGIN_INTEREST_INTERVAL` | `1m` | How often margin debt interest is catch-up accrued (O(1) per position) |
| `PRICE_DIFF_DB_PATH` | `data/pricediff.db` | SQLite for cross-exchange price difference watches/opportunities |
| `PRICE_DIFF_CHECK_INTERVAL` | `30s` | How often active price-diff watches are evaluated |
| `FUNDING_ARB_DB_PATH` | `data/fundingarb.db` | SQLite for funding-arb follow watches/signals |
| `FUNDING_ARB_CHECK_INTERVAL` | `30s` | How often active funding-arb watches are evaluated |
| `SCANNER_DB_PATH` | `data/scanner.db` | SQLite file for indicator scanner rules/results |
| `EXPORT_DB_PATH` | `data/export.db` | SQLite file for user data export jobs |
| `EXPORT_FILE_DIR` | `data/exports` | Directory for export download files |
| `EXPORT_FILE_TTL` | `1h` | How long completed export files remain downloadable |
| `EXPORT_WORKER_INTERVAL` | `2s` | How often pending exports are claimed |
| `IMPORT_DB_PATH` | `data/import.db` | SQLite file for user data import jobs |
| `IMPORT_FILE_DIR` | `data/imports` | Directory for uploaded import files |
| `IMPORT_FILE_TTL` | `1h` | How long preview uploads remain before cleanup |
| `IMPORT_WORKER_INTERVAL` | `2s` | How often pending imports are claimed |
| `ACCOUNT_DB_PATH` | `data/accounts.db` | SQLite for account close/reopen state |
| `ACCOUNT_PURGE_INTERVAL` | `1h` | How often expired closed accounts are purged |
| `FUTURES_HISTORY_DB_PATH` | `data/futures.db` | SQLite archive for OI, funding, long/short, liquidations |
| `FUTURES_HISTORY_INTERVAL` | `5m` | Snapshot worker cadence |
| `FUTURES_HISTORY_RETENTION` | `720h` | How long stored futures rows are kept |
| `FUTURES_HISTORY_SYMBOLS` | _(optional CSV)_ | Extra pairs always sampled (majors are built-in) |
| `ORDERBOOK_HISTORY_DB_PATH` | `data/orderbook.db` | SQLite archive for spot book samples |
| `ORDERBOOK_HISTORY_INTERVAL` | `1m` | Book snapshot worker cadence |
| `ORDERBOOK_HISTORY_RETENTION` | `168h` | How long stored books are kept |
| `ORDERBOOK_HISTORY_SYMBOLS` | _(optional CSV)_ | Extra pairs always sampled (majors are built-in) |
| `SCANNER_CHECK_INTERVAL` | `60s` | How often scanner rules are evaluated |

No API keys are required for the public endpoints used here. Respect upstream rate limits.

## MCP (AI tools) — same process as HTTP

MCP is **integrated into `cmd/server`**. Starting the backend exposes:

| Surface | URL |
|---------|-----|
| REST API | `http://localhost:8080/api/v1/...` |
| MCP (streamable HTTP) | `http://localhost:8080/mcp` |

```bash
go run ./cmd/server   # both REST and MCP
```

Package: `internal/transport/mcp` (in-process tools → market/watchlist services).  
Optional stdio-only binary: `SWYNGORA_API_URL=http://localhost:8080 API_AUTH_TOKEN=… go run ./cmd/mcp` (not required for normal use; `API_AUTH_TOKEN` or `SWYNGORA_API_TOKEN` is sent as `Authorization: Bearer` when the API is locked).

**Rule:** new agent-useful features must add MCP tools in the same change (root `AGENTS.md` §6.5).

## Test

```bash
go test ./...
go test ./internal/transport/http/ -count=1 -run TestE2E_
go test ./internal/transport/mcp/...
go vet ./...
```

Unit tests mock upstream HTTP; they do not call live Binance. `e2e_findings_test.go` hits the real router (`httptest`) for paper/auth regressions.

### Test coverage by layer (AGENTS.md §6.3 / §6.7)

| Layer | Package | Tests |
|---|---|---|
| Domain | `internal/domain` | `candle_test.go`, `errors_test.go`, `ports_test.go`, `ticker_test.go`, `supply_test.go`, `open_interest_test.go`, `volume_profile_test.go`, `absorption_test.go`, `liquidity_sweep_test.go`, `volume_surge_test.go`, `vwap_test.go`, `around_test.go`, `around_compare_test.go`, `around_moves_test.go`, `around_precursors_test.go`, `around_similar_test.go`, `pricediff_quote_test.go`, `funding_arb_test.go` |
| Application | `internal/service/market` | `service_test.go`, `volumeprofile_test.go`, `absorption_test.go`, `sweep_test.go`, `volumesurge_test.go`, `vwap_test.go`, `around_test.go`, `funding_arb_test.go` (fakes for ports) |
| Application | `internal/service/fundingarb` | `service_test.go` (create, min-profit notify, re-arm) |
| Infrastructure | `internal/adapter/fundingarbstore` | `sqlite_test.go` |
| Infrastructure | `internal/adapter/binance` | `client_test.go`, `supply_test.go`, `openinterest_test.go` (`httptest`) |
| Infrastructure | `internal/adapter/cache` | `ttl_test.go` |
| Infrastructure | `internal/adapter/watchliststore` | `memory_test.go`, `sqlite_test.go` (incl. reopen/restart persistence) |
| Infrastructure | `internal/adapter/alertstore` | `sqlite_test.go` (CRUD, one-shot trigger, reopen) |
| Application | `internal/service/pricealert` | `service_test.go` (validation, max, checker once-only) |
| Transport handlers | `internal/transport/http/handler` | `market_test.go`, `health_test.go`, `respond_test.go`, `pricediff_test.go`, `funding_arb_watch_test.go` |
| Transport middleware | `internal/transport/http/middleware` | `cors_test.go`, `ratelimit_test.go` |
| Transport router | `internal/transport/http` | `router_test.go` |
| Platform | `internal/platform/config` | `config_test.go` |
| Entrypoint | `cmd/server` | No unit tests — process wiring only; covered by router + adapter tests |

`cmd/server/main.go` is intentionally thin (wire deps + listen + graceful shutdown). Behavioral coverage lives in the layers it composes.

## Example curl

```bash
curl -s 'http://localhost:8080/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=3' | jq .
curl -s 'http://localhost:8080/api/v1/market/ticker/24h?symbol=BTCUSDT' | jq .
curl -s 'http://localhost:8080/api/v1/market/spot?quote=USDT&sort=quoteVolume&limit=5' | jq .
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
```
