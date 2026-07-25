# Feature: Market data (Binance candles, 24h volume, supply)

## Problem / goal

Expose first market-data APIs so clients can:

1. Load OHLCV candlesticks for a Binance pair at multiple intervals
2. Read 24-hour volume and price statistics
3. Read circulating supply for an asset (and market-cap columns on the spot list)

## Behavior

### Intervals — `GET /api/v1/market/intervals`

- Lists supported Binance candle intervals used by the candles endpoint.

### Product tags — `GET /api/v1/market/tags`

- Unique crypto tags from Binance product catalog (`get-products`), sorted
- Used for UI filters; excludes non-crypto tags (`bStocks`, `tCommodities`)
- Examples: `Meme`, `defi`, `Layer1_Layer2`, `AI`, `Payments`

### Exchanges — `GET /api/v1/market/exchanges`

- Venues: `binance` (default), `coinbase`, `bybit`
- Pass `exchange` on spot/candles/ticker/intervals/tags

### Spot markets — `GET /api/v1/market/spot`

- **Source:** Binance `GET /api/v3/exchangeInfo` + `GET /api/v3/ticker/24hr` (crypto spot only; tokenized equities `bStocks` and commodity wrappers `tCommodities` are excluded)
- **Params:**
  - `q` — search substring (symbol / base / quote / **tag name**)
  - `quote`, `base`, `status` — filters
  - `tag` / `tags` — Binance product-catalog tag filter (comma-separated or repeated; **OR** match, case-insensitive). Examples: `Meme`, `defi,AI`
  - `sort` — `quoteVolume` (default), `volume`, `priceChangePercent`, `lastPrice`, `tradeCount`, `symbol`, `baseAsset`, mcap fields, **`tags`**
  - `order` — `asc` | `desc` (default `desc` for metrics; `asc` for symbol/baseAsset/tags)
  - `limit` (default 50, max 500), `offset` (default 0)
- **Returns:** paged list with `total` match count, 24h metrics, **`tags`**, supply, and market caps
  - `marketCapCirculating` / `marketCapTotal` / `marketCapMax` = USD price × supply
  - `marketCapMax` is `"∞"` only when max supply is undefined **and** a USD price exists
  - Market-cap **sorts** collapse to one preferred quote per base (USDT > USDC > …); empty supply snapshot → `502`
  - Missing mcaps sort last (never treated as zero)
  - Supply/mcap is **not** fetched per user request: daily Binance marketing symbol-list refresh populates cache; requests are cache-only
  - base/quote/status are filter params only (not list columns)

### Candles — `GET /api/v1/market/candles`

- **Source:** Binance `GET /api/v3/klines`
- **Params:** `symbol` (required), `interval` (default `1h`), `limit` (default 100, max 1000), optional `startTime` / `endTime`
- **Intervals:** `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`

### 24h ticker / volume — `GET /api/v1/market/ticker/24h`

- **Source:** Binance `GET /api/v3/ticker/24hr`
- **Params:** `symbol` (required)
- **Returns:** last/open/high/low, base `volume`, quote `quoteVolume`, change %, trade count

### Supply — `GET /api/v1/market/supply`

- **Source:** Binance marketing symbol list (`circulatingSupply`, `totalSupply`, `maxSupply`); not Spot REST kline/ticker APIs
- **Params:** `asset` or `symbol` (e.g. `BTC` or `BTCUSDT`)
- **Returns:** circulating / total / max (max null when undefined), optional USD price, `source: binance`

### Caching

In-memory TTL caches with periodic cleanup:

| Data | Default TTL |
|---|---|
| Candles | 30s (latest-N only; max 512 keys; ranges uncached) |
| Spot markets (joined list) | 5s (default) |
| Supply snapshot | Daily @ 03:00 UTC (default) + startup; atomic replace; default never-expire last-good |
| Ticker | 15s |

## Code locations

| Area | Path |
|---|---|
| Domain | `backend/internal/domain/` |
| Use cases | `backend/internal/service/market/` |
| Binance adapter | `backend/internal/adapter/binance/` (market + supply) |
| HTTP | `backend/internal/transport/http/` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| Test UI | `simple-frontend/` |

## How to verify

```bash
cd backend && go test ./...
go run ./cmd/server
# another terminal
curl -s 'http://localhost:8080/api/v1/market/supply?asset=BTC' | jq .
curl -s 'http://localhost:8080/api/v1/market/spot?quote=USDT&limit=20' | jq '.items[] | {symbol, circulatingSupply, marketCapCirculating}'
```

## Known limitations

- Marketing symbol list is a public web API; response shape can change — adapter isolation + tests required
- Coverage is Binance-listed marketing symbols (~470), not every historical exchangeInfo pair
- Fiat pairs (e.g. EUR) are not crypto supply
- Max supply is null for coins without a hard cap (e.g. ETH)
