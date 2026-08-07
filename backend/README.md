# Swyngora Backend

Go HTTP API for market data across **Binance**, **Coinbase**, and **Bybit** (spot). Circulating supply remains from the Binance daily snapshot (asset-level mcap enrichment for all venues).

## Architecture (N-layered)

| Layer | Path | Role |
|---|---|---|
| Transport | `internal/transport/http` | HTTP handlers, CORS, rate limit, JSON mapping |
| Transport | `internal/transport/telegram` | Optional Telegram bot (long-poll → same services) |
| Application | `internal/service/market` | Validation + use-case orchestration |
| Domain | `internal/domain` | Entities, ports, sentinel errors |
| Infrastructure | `internal/adapter/*` | Binance (market + supply), TTL cache, **SQLite watchlist** |
| Platform | `internal/platform/config` | Env config |

OpenAPI contract: [`api/openapi/openapi.yaml`](api/openapi/openapi.yaml).

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/api/v1/market/exchanges` | Supported venues |
| `GET` | `/api/v1/market/intervals` | Candle intervals (per `exchange`) |
| `GET` | `/api/v1/market/tags` | Unique Binance product-catalog tags (crypto) |
| `GET` | `/api/v1/market/spot?q=btc&quote=USDT&tag=Meme&sort=quoteVolume` | List/search/filter/sort spot markets |
| `GET` | `/api/v1/market/indicators?symbol=BTCUSDT&interval=1h` | RSI (Wilder) + EMA series |
| `POST` | `/api/v1/market/indicators/batch` | Latest RSI/EMA for up to 50 symbols (bounded concurrency) |
| `GET` | `/api/v1/watchlist` | Get watchlist + `version` (`clientId` / optional `ownerClientId`) |
| `POST` | `/api/v1/watchlist/items` | Add symbol (owner or editor; optional `baseVersion`) |
| `DELETE` | `/api/v1/watchlist/items?exchange=&symbol=` | Remove symbol (`baseVersion` / If-Match) |
| `PUT` | `/api/v1/watchlist` | Replace list (owner; `baseVersion` + optional `baseItems` for multi-device) |
| `GET`/`POST`/`PATCH`/`DELETE` | `/api/v1/watchlist/shares` | List / grant / update role / revoke shares |
| `GET` | `/api/v1/watchlist/shared` | Lists shared with the caller |
| `GET` | `/api/v1/watchlist/audit` | Change history (who/when) |
| `GET` | `/api/v1/alerts` | List price alerts (`clientId` / `X-Client-Id`) |
| `POST` | `/api/v1/alerts` | Create one-shot above/below price alert |
| `GET` | `/api/v1/alerts/{id}` | Get one alert |
| `DELETE` | `/api/v1/alerts/{id}` | Delete an alert |
| `GET`/`PUT`/`DELETE` | `/api/v1/alerts/webhook` | Get / set / clear alert webhook URL |
| `POST` | `/api/v1/portfolio` | Create paper portfolio (starting balance) |
| `GET` | `/api/v1/portfolio` | Cash, positions, realized/unrealized P&L |
| `POST` | `/api/v1/portfolio/orders` | Paper market or pending (`limit_buy` / `limit_sell` / `stop_loss`) |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (default: open) |
| `GET` | `/api/v1/portfolio/orders/{id}` | One pending order + last price + amend hints |
| `PATCH` | `/api/v1/portfolio/orders/{id}` | Amend open GTC limit/stop price and/or remaining size |
| `POST` | `/api/v1/portfolio/orders/cancel-all` | Cancel all open paper orders, or one market (`symbol`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel an open pending order |
| `GET` | `/api/v1/portfolio/trades` | Paper trade history |
| `POST`/`GET` | `/api/v1/portfolio/recurring-buys` | Create / list paper recurring buy (DCA) plans |
| `GET`/`DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Get / delete plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Recurring buy execution history |
| `POST`/`GET` | `/api/v1/portfolio/margin/orders` | Paper margin open (market/limit long/short 1x–10x) / list |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel margin limit order |
| `GET` | `/api/v1/portfolio/margin/positions` | Open margin positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | Get margin position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full or partial margin close |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Stop-loss / take-profit |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |
| `POST`/`GET` | `/api/v1/price-diff/watches` | Create / list cross-exchange price difference watches |
| `GET`/`DELETE` | `/api/v1/price-diff/watches/{id}` | Get / delete watch |
| `GET` | `/api/v1/price-diff/opportunities` | List opportunities (`status=open\|closed\|all`) |
| `GET` | `/api/v1/price-diff/opportunities/{id}` | Get opportunity |
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
| `POST` | `/api/v1/export` | Start JSON/CSV export of own watchlist/shares/alerts/backtests |
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
| `GET` | `/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=100` | OHLCV from Binance |
| `GET` | `/api/v1/market/ticker/24h?symbol=BTCUSDT` | 24h stats + base/quote volume |
| `GET` | `/api/v1/market/supply?asset=BTC` | Circulating supply (Binance product catalog) |

Intervals: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`.

Optional candle params: `startTime`, `endTime` (RFC3339 or Unix ms).

**Supply / mcap note:** Circulating / total / max supply is loaded from Binance’s public **marketing symbol list** on a **daily schedule** (default **03:00 UTC**, plus once on startup). Failed refreshes **retry with backoff** (1m→1h) before waiting for the next daily slot. User requests (`/supply`, spot mcap columns) **read from cache only**. Snapshots are **atomically replaced** on successful refresh (last-good retained on failure until safety TTL). Default **`SUPPLY_CACHE_TTL=48h`** so entries cannot live forever after a long outage (set `0` to never expire). Max is null when Binance does not define a hard cap (max mcap may show as infinite **only when a USD price exists**). Sorting by market-cap fields **collapses to one preferred quote pair per base** (USDT-first). Empty supply snapshot → market-cap sort returns `502`.

**Watchlist persistence:** client watchlists are stored in **SQLite** (default `data/watchlist.db`) so they survive process restarts. Configure path via `WATCHLIST_DB_PATH`. **Sharing:** owners grant `viewer` or `editor` access to other `clientId`s; editors may add/remove symbols only; all mutations write an audit log (`docs/features/watchlist-sharing.md`). **Multi-device sync:** each list has a monotonic `version`; send `baseVersion` on writes — non-overlapping adds auto-merge; delete-vs-update conflicts return **409** with both sides (`docs/features/watchlist-sync.md`).

**Price alerts:** above/below thresholds (`POST /api/v1/alerts`) with `mode=one_time` or `mode=repeating`. Optional webhook (`/api/v1/alerts/webhook`) supports `deliveryMode=immediate` or `hourly_digest`, plus **quiet hours** (`timeZone` + local start/end; midnight-crossing ranges OK). Delivery waits until quiet hours end; pending rows survive restarts.

**Cross-exchange price diff:** watches (`/api/v1/price-diff/watches`) compare last prices on Binance, Coinbase, and Bybit after fees; opportunities record buy/sell venues when net edge exceeds `minNetDiffPct`. Open state is durable; no duplicate while open; re-opens after the edge drops and returns. Stale/missing prices skip that venue. Interval `PRICE_DIFF_CHECK_INTERVAL` (default `30s`).

**Paper trading:** virtual portfolio (`/api/v1/portfolio`) with starting cash, market buy/sell at last price, pending limit/stop orders with cash/position **reservations**, **partial fills**, **in-place amend** of open GTC limit/stop (`PATCH .../orders/{id}`), and **GTC/IOC/FOK** (+ optional GTC `expiresAt`) via the background filler, open positions, realized/unrealized P&L, trade history, **recurring buy (DCA) plans**, and **isolated margin** long/short (1x–10x, market/limit, liquidation, partial close, SL/TP). Simulated only — not real money. SQLite path `PORTFOLIO_DB_PATH` (default `data/portfolio.db`); order check interval `PORTFOLIO_ORDER_CHECK_INTERVAL` (default `15s`); recurring buy interval `RECURRING_BUY_INTERVAL` (default `30s`).

**Indicator scanner:** create RSI / EMA crossover / volume-increase rules for the client's watchlist (`/api/v1/scanner/rules`). A background job evaluates rules on `SCANNER_CHECK_INTERVAL` (default `60s`), writes matches to history (`/api/v1/scanner/results`), and skips duplicates for the same rule + symbol + candle (`marketDataKey`). **Historical backtests** (`/api/v1/scanner/backtests`) re-run a rule over a date range for one symbol, track progress, support cancel, and report 1/5/20-day forward returns per signal. SQLite path `SCANNER_DB_PATH` (default `data/scanner.db`).

**User data export:** `POST /api/v1/export` queues a JSON or CSV dump of the caller's watchlist, shares, alerts, and backtests. One active job per client; poll progress; cancel supported; download is owner-only; files expire (`EXPORT_FILE_TTL`, default 1h). See `docs/features/user-data-export.md`.

**User data import:** `POST /api/v1/import/preview` uploads a prior export and returns valid/invalid/willAdd counts; `confirm` with `merge` or `replace` applies in the background with progress/cancel and dedupe. See `docs/features/user-data-import.md`.

**Account close:** `POST /api/v1/account/close` closes a `clientId` for 7 days (reopen allowed); product APIs and shared-list access stop; after grace, watchlists/shares/alerts/backtests/import-export files and jobs are purged. See `docs/features/account-close.md`.

**Hardening:** per-IP rate limits with **capped bucket map**; sanitized public errors; candle/ticker singleflight; bounded candle + watchlist client maps; non-crypto product filter **fails closed** without last-good catalog (no equities/commodities as crypto); indicator batch uses process-wide upstream semaphore.

## Run

```bash
# from backend/
go run ./cmd/server
# listens on :8080 by default
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

Enabled when `TELEGRAM_BOT_TOKEN` is non-empty. Calls **market** and **watchlist** services in-process (no HTTP hop).

| Command | Description |
|---------|-------------|
| `/price <symbol> [exchange]` | 24h ticker |
| `/spot [exchange] [query]` | Top by quote volume |
| `/lowmcap [exchange\|all] [n]` | Lowest circulating market cap |
| `/mcap <asset\|pair>` | Supply snapshot |
| `/rsi <symbol> [interval] [exchange]` | RSI + EMA |
| `/exchanges` | Venues |
| `/watch` · `add` · `del` · `top` | Per-user watchlist (`tg-<user_id>`) |

See [`docs/features/telegram-bot.md`](../docs/features/telegram-bot.md).

### Environment

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Listen address |
| `TELEGRAM_BOT_TOKEN` | _(empty = disabled)_ | BotFather token; enables Telegram transport |
| `TELEGRAM_CHAT_ID` | — | Allowed chat id (or use `TELEGRAM_ALLOWED_CHAT_IDS`) |
| `TELEGRAM_ALLOWED_CHAT_IDS` | — | Comma-separated allowed chat ids |
| `TELEGRAM_ALLOW_ALL` | `false` | If true and allowlist empty, bot is public; otherwise token without allowlist **does not start** |
| `CORS_ALLOW_ORIGINS` | `*` | Comma-separated browser origins (`*` = any; set exact origins in production) |
| `BOT_DEFAULT_EXCHANGE` | `binance` | Default venue for bot commands |
| `BOT_POLL_TIMEOUT` | `30s` | Telegram long-poll wait |
| `BOT_LOWMCAP_LIMIT` | `10` | Default `/lowmcap` size (max 25) |
| `BINANCE_BASE_URL` | `https://api.binance.com` | Binance Spot REST base |
| `BINANCE_PRODUCT_BASE_URL` | `https://www.binance.com` | Host for marketing symbol list (supply) |
| `COINBASE_BASE_URL` | `https://api.coinbase.com` | Coinbase public market products |
| `COINBASE_EXCHANGE_URL` | `https://api.exchange.coinbase.com` | Coinbase public candles |
| `BYBIT_BASE_URL` | `https://api.bybit.com` | Bybit v5 public market API |
| `HTTP_CLIENT_TIMEOUT` | `15s` | Upstream HTTP timeout |
| `CANDLE_CACHE_TTL` | `30s` | Candle response TTL (latest-N queries only; ranges not cached) |
| `CANDLE_CACHE_MAX_ENTRIES` | `512` | Max candle cache keys (memory bound) |
| `TICKER_CACHE_TTL` | `15s` | Ticker response TTL |
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
| `MARGIN_INTEREST_INTERVAL` | `1m` | How often margin debt interest is catch-up accrued (O(1) per position) |
| `PRICE_DIFF_DB_PATH` | `data/pricediff.db` | SQLite for cross-exchange price difference watches/opportunities |
| `PRICE_DIFF_CHECK_INTERVAL` | `30s` | How often active price-diff watches are evaluated |
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
Optional stdio-only binary: `go run ./cmd/mcp` (not required for normal use).

**Rule:** new agent-useful features must add MCP tools in the same change (root `AGENTS.md` §6.5).

## Test

```bash
go test ./...
go test ./internal/transport/mcp/...
go vet ./...
```

Unit tests mock upstream HTTP; they do not call live Binance.

### Test coverage by layer (AGENTS.md §6.3 / §6.7)

| Layer | Package | Tests |
|---|---|---|
| Domain | `internal/domain` | `candle_test.go`, `errors_test.go`, `ports_test.go`, `ticker_test.go`, `supply_test.go` |
| Application | `internal/service/market` | `service_test.go` (fakes for ports) |
| Infrastructure | `internal/adapter/binance` | `client_test.go`, `supply_test.go` (`httptest`) |
| Infrastructure | `internal/adapter/cache` | `ttl_test.go` |
| Infrastructure | `internal/adapter/watchliststore` | `memory_test.go`, `sqlite_test.go` (incl. reopen/restart persistence) |
| Infrastructure | `internal/adapter/alertstore` | `sqlite_test.go` (CRUD, one-shot trigger, reopen) |
| Application | `internal/service/pricealert` | `service_test.go` (validation, max, checker once-only) |
| Transport handlers | `internal/transport/http/handler` | `market_test.go`, `health_test.go`, `respond_test.go` |
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
