# Swyngora Backend

Go HTTP API for market data across **Binance**, **Coinbase**, and **Bybit** (spot). Circulating supply remains from the Binance daily snapshot (asset-level mcap enrichment for all venues).

## Architecture (N-layered)

| Layer | Path | Role |
|---|---|---|
| Transport | `internal/transport/http` | HTTP handlers, CORS, rate limit, JSON mapping |
| Application | `internal/service/market` | Validation + use-case orchestration |
| Domain | `internal/domain` | Entities, ports, sentinel errors |
| Infrastructure | `internal/adapter/*` | Binance (market + supply), TTL cache |
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
| `GET` | `/api/v1/watchlist` | Get watchlist (`clientId` / `X-Client-Id`) |
| `POST` | `/api/v1/watchlist/items` | Add symbol to watchlist |
| `DELETE` | `/api/v1/watchlist/items?exchange=&symbol=` | Remove from watchlist |
| `PUT` | `/api/v1/watchlist` | Replace entire watchlist |
| `GET` | `/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=100` | OHLCV from Binance |
| `GET` | `/api/v1/market/ticker/24h?symbol=BTCUSDT` | 24h stats + base/quote volume |
| `GET` | `/api/v1/market/supply?asset=BTC` | Circulating supply (Binance product catalog) |

Intervals: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`.

Optional candle params: `startTime`, `endTime` (RFC3339 or Unix ms).

**Supply / mcap note:** Circulating / total / max supply is loaded from Binance’s public **marketing symbol list** on a **daily schedule** (default **03:00 UTC**, plus once on startup). Failed refreshes **retry with backoff** (1m→1h) before waiting for the next daily slot. User requests (`/supply`, spot mcap columns) **read from cache only**. Snapshots are **atomically replaced** on successful refresh (last-good retained on failure until safety TTL). Default **`SUPPLY_CACHE_TTL=48h`** so entries cannot live forever after a long outage (set `0` to never expire). Max is null when Binance does not define a hard cap (max mcap may show as infinite **only when a USD price exists**). Sorting by market-cap fields **collapses to one preferred quote pair per base** (USDT-first). Empty supply snapshot → market-cap sort returns `502`.

**Hardening:** per-IP rate limits with **capped bucket map**; sanitized public errors; candle/ticker singleflight; bounded candle + watchlist client maps; non-crypto product filter **fails closed** without last-good catalog (no equities/commodities as crypto); indicator batch uses process-wide upstream semaphore.

## Run

```bash
# from backend/
go run ./cmd/server
# listens on :8080 by default
```

### Environment

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Listen address |
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

No API keys are required for the public endpoints used here. Respect upstream rate limits.

## Test

```bash
go test ./...
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
