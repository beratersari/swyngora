# Swyngora Backend

Go HTTP API for market data. First slice: **Binance** candles + 24h volume/ticker, and **asset supply** (circulating / total / max) from free CoinGecko metadata.

## Architecture (N-layered)

| Layer | Path | Role |
|---|---|---|
| Transport | `internal/transport/http` | HTTP handlers, CORS, JSON mapping |
| Application | `internal/service/market` | Validation + use-case orchestration |
| Domain | `internal/domain` | Entities, ports, sentinel errors |
| Infrastructure | `internal/adapter/*` | Binance, CoinGecko, TTL cache |
| Platform | `internal/platform/config` | Env config |

OpenAPI contract: [`api/openapi/openapi.yaml`](api/openapi/openapi.yaml).

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/api/v1/market/intervals` | Supported candle intervals |
| `GET` | `/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=100` | OHLCV from Binance |
| `GET` | `/api/v1/market/ticker/24h?symbol=BTCUSDT` | 24h stats + base/quote volume |
| `GET` | `/api/v1/market/supply?asset=BTC` | Circulating / total / max supply |

Intervals: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`.

Optional candle params: `startTime`, `endTime` (RFC3339 or Unix ms).

**Supply note:** Binance public market APIs do not expose circulating/total/max supply. The supply endpoint uses the free CoinGecko public API and is labeled in the response `source` / `note` fields.

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
| `BINANCE_BASE_URL` | `https://api.binance.com` | Binance REST base |
| `COINGECKO_BASE_URL` | `https://api.coingecko.com` | CoinGecko REST base |
| `HTTP_CLIENT_TIMEOUT` | `15s` | Upstream HTTP timeout |
| `CANDLE_CACHE_TTL` | `30s` | Candle response TTL |
| `TICKER_CACHE_TTL` | `15s` | Ticker response TTL |
| `SUPPLY_CACHE_TTL` | `5m` | Supply response TTL |
| `CACHE_CLEANUP_EVERY` | `1m` | Background expired-entry cleanup |

No API keys are required for the public endpoints used here. Respect upstream rate limits.

## Test

```bash
go test ./...
go vet ./...
```

Unit tests mock upstream HTTP; they do not call live Binance/CoinGecko.

### Test coverage by layer (AGENTS.md §6.3 / §6.7)

| Layer | Package | Tests |
|---|---|---|
| Domain | `internal/domain` | `candle_test.go`, `errors_test.go`, `ports_test.go`, `ticker_test.go`, `supply_test.go` |
| Application | `internal/service/market` | `service_test.go` (fakes for ports) |
| Infrastructure | `internal/adapter/binance` | `client_test.go` (`httptest`) |
| Infrastructure | `internal/adapter/coingecko` | `client_test.go` (`httptest`) |
| Infrastructure | `internal/adapter/cache` | `ttl_test.go` |
| Transport handlers | `internal/transport/http/handler` | `market_test.go`, `health_test.go`, `respond_test.go` |
| Transport middleware | `internal/transport/http/middleware` | `cors_test.go` |
| Transport router | `internal/transport/http` | `router_test.go` |
| Platform | `internal/platform/config` | `config_test.go` |
| Entrypoint | `cmd/server` | No unit tests — process wiring only; covered by router + adapter tests |

`cmd/server/main.go` is intentionally thin (wire deps + listen + graceful shutdown). Behavioral coverage lives in the layers it composes.

## Example curl

```bash
curl -s 'http://localhost:8080/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=3' | jq .
curl -s 'http://localhost:8080/api/v1/market/ticker/24h?symbol=BTCUSDT' | jq .
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
```
