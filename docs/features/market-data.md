# Feature: Market data (Binance candles, 24h volume, supply)

## Problem / goal

Expose first market-data APIs so clients can:

1. Load OHLCV candlesticks for a Binance pair at multiple intervals
2. Read 24-hour volume and price statistics
3. Read circulating / total / max supply for an asset

## Behavior

### Intervals — `GET /api/v1/market/intervals`

- Lists supported Binance candle intervals used by the candles endpoint.

### Candles — `GET /api/v1/market/candles`

- **Source:** Binance `GET /api/v3/klines`
- **Params:** `symbol` (required), `interval` (default `1h`), `limit` (default 100, max 1000), optional `startTime` / `endTime`
- **Intervals:** `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`

### 24h ticker / volume — `GET /api/v1/market/ticker/24h`

- **Source:** Binance `GET /api/v3/ticker/24hr`
- **Params:** `symbol` (required)
- **Returns:** last/open/high/low, base `volume`, quote `quoteVolume`, change %, trade count

### Supply — `GET /api/v1/market/supply`

- **Source:** CoinGecko free public API (not Binance — supply is not on Binance public market endpoints)
- **Params:** `asset` or `symbol` (e.g. `BTC` or `BTCUSDT`)
- **Returns:** circulatingSupply, totalSupply, maxSupply (nullable), optional USD price, `source: coingecko`

### Caching

In-memory TTL caches with periodic cleanup:

| Data | Default TTL |
|---|---|
| Candles | 30s |
| Ticker | 15s |
| Supply | 5m |

## Code locations

| Area | Path |
|---|---|
| Domain | `backend/internal/domain/` |
| Use cases | `backend/internal/service/market/` |
| Binance adapter | `backend/internal/adapter/binance/` |
| CoinGecko adapter | `backend/internal/adapter/coingecko/` |
| HTTP | `backend/internal/transport/http/` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| Test UI | `simple-frontend/` |

## How to verify

```bash
cd backend && go test ./...
go run ./cmd/server
# another terminal
curl -s 'http://localhost:8080/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=3'
curl -s 'http://localhost:8080/api/v1/market/ticker/24h?symbol=BTCUSDT'
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC'
```

Or open `simple-frontend/` via a static server and use **Fetch all**.

## Errors

| HTTP | Code | When |
|---|---|---|
| 400 | `invalid_argument` | Missing/invalid params |
| 404 | `not_found` | Unknown symbol/asset |
| 429 | `rate_limited` | Upstream rate limit / ban |
| 502 | `upstream_error` | Upstream failure / bad payload |

## Known limitations

- No WebSocket streaming yet (REST only)
- Supply depends on CoinGecko rate limits and free-tier availability
- No auth; public read-only endpoints for local/dev
- Single exchange (Binance) for market data
