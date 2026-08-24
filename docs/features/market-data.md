# Feature: Market data (Binance candles, 24h volume, supply)

## Problem / goal

Expose first market-data APIs so clients can:

1. Load OHLCV candlesticks for a Binance pair at multiple intervals
2. Read 24-hour volume and price statistics
3. Read circulating supply for an asset (and market-cap columns on the spot list)

## Behavior

### Intervals — `GET /api/v1/market/intervals`

- Lists supported candle intervals for `?exchange=` (Binance / Coinbase / Bybit sets differ).

### Product tags — `GET /api/v1/market/tags`

- Unique crypto tags from Binance product catalog (`get-products`), sorted
- Optional `exchange` (tags only for Binance; other venues return empty)
- Used for UI filters; excludes non-crypto tags (`bStocks`, `tCommodities`)
- Examples: `Meme`, `defi`, `Layer1_Layer2`, `AI`, `Payments`

### Display FX — `GET /api/v1/market/fx`

- USD-based spot rates (units of currency per 1 USD) from the free Frankfurter/ECB feed
- Always includes `USD=1` and `USDT=1` (USDT treated as USD)
- Cached 15 minutes; `stale=true` when last-good is served after an upstream miss
- Display conversion only — not settlement FX. Web header switcher converts BIST TRY, Nasdaq/Coinbase USD, and crypto USDT last/open/high/low/quote volume/mcap/chart OHLC

### Exchanges — `GET /api/v1/market/exchanges`

- Venues: `binance` (default), `coinbase`, `bybit`, **`nasdaq`** (US stocks, USD), **`bist`** (Borsa Istanbul, TRY)
- Pass `exchange` on spot/candles/ticker/intervals/tags
- **Product tags** come from the Binance marketing catalog and are **applied cross-venue by base asset** for **crypto** venues only (Coinbase/Bybit rows get the same tags as Binance when the base matches). Nasdaq/BIST never inherit crypto tags — `LINK` on BIST is Link Bilgisayar, not Chainlink.
- **24h trade count** is only available from Binance public APIs; Coinbase/Bybit return 0 / UI shows "—"
- **Coinbase high/low:** Advanced Trade public `products` leaves `high_24h`/`low_24h` empty; detail ticker fills them from Exchange `GET /products/{id}/stats`

### Price heatmap (web)

`/heatmap` is a CoinMarketCap-style treemap of `GET /api/v1/market/spot`. Red / slate / green tiles with a 2px white gap and 2px corner radius (CoinMarketCap spacing); `|Δ| < 0.05%` is dead-band gray. Cell size is circulating market cap (default) or 24h volume. Hover tooltip shows price, market cap, 24h volume, and venue; click opens coin detail. Fullscreen expands the map.

Coin detail also has an **order heatmap** (resting bid/ask size over time) from `GET /api/v1/market/orderbook/heatmap`. That is not the market-cap treemap — see [`order-book.md`](order-book.md).

### Spot order book — `GET /api/v1/market/orderbook`

- Grouped bid/ask depth (spot only). `group` is the price step (e.g. `0.1`); omit for a suggested default.
- `limit` is grouped rows per side. Backend marks `isWall` on unusually large buckets.
- Live local books on Binance, Coinbase, and Bybit (websocket; gap/drop resyncs).
- `analysis` uses depth within ±`rangePct` of mid (default 2%) for pressure, imbalance, and walls. Walls include `behavior` (`short` / `persistent` / `suspicious`).
- `GET /api/v1/market/orderbook/combined` sums all three venues in a symmetric ±% both sides can reach.
- `GET /api/v1/market/orderbook/impact` walks live asks (buy) or bids (sell) for a size (`quantity` or `notional`) and returns average fill, slippage, and touch impact when leftover depth remains. If the visible side is wiped, impact is not calculated.
- `GET /api/v1/market/orderbook/liquidity` scores 0–100 from ±0.1 / ±0.5 / ±1% bid/ask notional that the book actually covers; market-wide uses the common venue range.
- `GET /api/v1/market/liquidations` rolling 5m/1h/4h/24h long vs short futures liquidations (Binance USD-M + Bybit linear). See [`liquidations.md`](liquidations.md).
- `GET /api/v1/market/open-interest` current futures open interest plus 5m/1h/4h/24h change (Binance USD-M + Bybit linear). Includes `funding`. See [`open-interest.md`](open-interest.md).
- `GET /api/v1/market/funding-rate` predicted next perpetual funding rate plus recent settlements. See [`funding-rate.md`](funding-rate.md).
- `GET /api/v1/market/long-short-ratio` account long/short ratio plus recent 5m history. See [`long-short-ratio.md`](long-short-ratio.md).
- `GET /api/v1/market/futures-history` durable stored OI / funding / long-short / liquidation rows. See [`futures-history.md`](futures-history.md).
- `GET /api/v1/market/liquidation-hunt` hypothetical per-venue hunt (spot size to reach estimated liq zones + rough desk result). See [`liquidation-hunt.md`](liquidation-hunt.md).
- `GET /api/v1/market/squeeze-risk` long/short squeeze risk scores per venue + combined. See [`squeeze-risk.md`](squeeze-risk.md).
- `GET /api/v1/market/positioning` price + open-interest regime (buildup / unwinding / covering) per venue + combined. See [`positioning.md`](positioning.md).
- `GET /api/v1/market/venue-divergence` Binance vs Bybit same/opposite split. See [`venue-divergence.md`](venue-divergence.md).
- `GET /api/v1/market/taker-flow` aggressive futures buy vs sell volume (5m/1h/4h). See [`taker-flow.md`](taker-flow.md).
- `GET /api/v1/market/cvd` spot and futures CVD versus price (15m / 1h / 4h / 24h); venue split and spot-vs-futures split. See [`cvd.md`](cvd.md).
- `GET /api/v1/market/volume-profile` volume by price (POC + 70% value area, buy/sell); Binance and Bybit separately plus combined. See [`volume-profile.md`](volume-profile.md).
- `GET /api/v1/market/vwap` volume-weighted average price from a start time; last vs VWAP; Binance, Bybit, combined. See [`vwap.md`](vwap.md).
- `GET /api/v1/market/absorption` large market buys/sells versus little price move; which side is absorbing and how strong. See [`absorption.md`](absorption.md).
- `GET /api/v1/market/liquidity-sweeps` poke through a prior high/low that comes back; level, excursion, reclaim time, volume. See [`liquidity-sweeps.md`](liquidity-sweeps.md).
- `GET /api/v1/market/volume-surge` current 5m/15m/1h volume vs that coin's typical; buy/sell split. See [`volume-surge.md`](volume-surge.md).
- `GET /api/v1/market/volume-surge/scan` rank coins whose volume is much higher than typical. See [`volume-surge.md`](volume-surge.md).
- `GET /api/v1/market/basis` perp vs spot/index premium or discount. See [`basis.md`](basis.md).
- `GET /api/v1/market/correlation` how similarly a coin moves with BTC and ETH (1h / 4h / 24h). See [`correlation.md`](correlation.md).
- `GET /api/v1/market/breadth` how many followed coins are up vs down (1h / 4h / 24h). See [`breadth.md`](breadth.md).
- `GET /api/v1/market/volatility` how much a coin moved (range, vs normal, vs BTC/ETH). See [`volatility.md`](volatility.md).
- `GET /api/v1/market/snapshot` price, volume, mcap, OI, funding, LS, and taker flow together. See [`snapshot.md`](snapshot.md).
- `GET /api/v1/market/levels` support and resistance plus breakout strength. See [`levels.md`](levels.md).
- `GET /api/v1/market/whales` large trades and liquidations, biggest first. See [`whales.md`](whales.md).
- `GET /api/v1/market/orderbook/history` stored book at a time (or a list). See [`book-history.md`](book-history.md).
- `GET /api/v1/market/orderbook/history/compare` which levels gained or lost liquidity. See [`book-history.md`](book-history.md).
- `GET /api/v1/market/orderbook/icebergs` same-price clip eaten then refilled. See [`icebergs.md`](icebergs.md).
- See [`order-book.md`](order-book.md).

### Equities — `nasdaq` / `bist`

Cash stocks use the same list/ticker/candle contracts as crypto pairs:

| Venue | Quote | Symbols | Source |
|-------|-------|---------|--------|
| `nasdaq` | USD | Full Nasdaq tape (~4k) | Nasdaq.com public screener (last, % change, volume, **market cap**) |
| `bist` | TRY | Live BIST board (~640) | TradingView public Turkey scanner (last, % change, volume, **TRY market cap**, sector). Fallback: Bigpara list + Yahoo spark (no mcap). |

Metrics: last, session change %, day high/low (Yahoo/spark on detail), share volume, notional volume (`price × volume`). Nasdaq **market cap** comes from the Nasdaq screener. BIST **market cap** comes from the Turkey scanner (TRY). Yahoo spark (used for candles and the BIST fallback tape) does **not** publish mcap. No public order book (detail book is empty / 404).

Ticker and candle fetches **fail closed** when Yahoo is down — expired last-good quotes are not returned as live (paper fills and price alerts use last price as current). Spot list may serve last-good after a screener miss; those caches are cleaned on the same process TTL tick as crypto.

### Spot markets — `GET /api/v1/market/spot`

- **Source (crypto):** Binance `GET /api/v3/exchangeInfo` + `GET /api/v3/ticker/24hr` (crypto spot only; tokenized equities `bStocks` and commodity wrappers `tCommodities` are excluded). Product catalog is **required** for the filter: without a warm or stale catalog snapshot, the spot list returns `502` rather than listing non-crypto products.
- **Source (nasdaq/bist):** Nasdaq.com screener and the public TradingView Turkey scanner. Market-cap sort uses the venue mcap field (no Binance supply snapshot — crypto circulating supply is never applied to stocks).
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
  - Supply/mcap is **not** fetched per user request: daily Binance marketing symbol-list refresh populates cache; requests are cache-only. After TTL the last-good snapshot is still served with `stale: true` (supply/catalog are not evicted on cleanup). Unmapped assets return `404` `supply_unmapped`.
  - base/quote/status are filter params only (not list columns)

### Candles — `GET /api/v1/market/candles`

- **Source:** venue klines (Binance / Coinbase / Bybit)
- **Params:** `exchange`, `symbol` (required), `interval` (default `1h`), `limit` (default 100, max 1000), optional `startTime` / `endTime`
- **Intervals:** exchange-specific — use `/intervals?exchange=`; Binance supports the full set (`1m`…`1M`); Coinbase/Bybit are stricter

### 24h ticker / volume — `GET /api/v1/market/ticker/24h`

- **Source:** venue 24h ticker APIs
- **Params:** `exchange`, `symbol` (required)
- **Returns:** last/open/high/low, base `volume`, quote `quoteVolume`, change %, trade count

### Holders — `GET /api/v1/market/holders`

- **Source:** CoinMarketCap public `data-api` detail, mapped by Binance marketing `cmcUniqueId`
- **Params:** `asset` or `symbol` (e.g. `BTC` or `BTCUSDT`)
- **Returns:** holder count, optional daily-active, top 10/20/50/100 share %, up to 20 wallets
- Cached 1h; 404 when unpublished. See [`holders.md`](holders.md).

### Supply — `GET /api/v1/market/supply`

- **Source:** Binance marketing symbol list (`circulatingSupply`, `totalSupply`, `maxSupply`); not Spot REST kline/ticker APIs
- **Params:** `asset` or `symbol` (e.g. `BTC` or `BTCUSDT`)
- **Returns:** circulating / total / max (max null when undefined), optional USD price, `source: binance`

### Indicators batch — `POST /api/v1/market/indicators/batch`

- Body: `{ exchange, interval, symbols[], rsiPeriod?, emaPeriods? }` (max 50 symbols)
- Returns latest RSI/EMA per symbol; per-symbol failures redacted as `error: unavailable`
- Process-wide upstream concurrency cap (plus per-request limit)

### Caching

In-memory TTL caches with periodic cleanup:

| Data | Default TTL |
|---|---|
| Candles | 30s (latest-N only; max 512 keys; ranges uncached) |
| Spot markets (joined list) | 5s (default) |
| Supply snapshot | Daily @ 03:00 UTC + startup + failure retries; atomic replace; **48h safety TTL** (env) |
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


### Indicators — `GET /api/v1/market/indicators`

- Computes **RSI** (Wilder's smoothing, default period 14) and **EMA** (default 12, 26) from exchange candles
- Params: `exchange`, `symbol`, `interval`, `limit`, `rsiPeriod`, `emaPeriods` (comma-separated integers)
- Invalid closes fail the request (gaps are not collapsed — that would corrupt RSI/EMA)
- Out-of-range `emaPeriods` are rejected (not silently defaulted)
- Returns point series + `latest` snapshot; warm-up bars may have null indicator values
- **Not financial advice** — informational analysis only

### Watchlist — `/api/v1/watchlist`

- **No server auth** (demo only): scoped by client-supplied `clientId` query or `X-Client-Id` header
- `clientId` is **required** (non-empty); the shared name `default` is rejected
- Simple frontend generates an unguessable browser id in `localStorage`
- `GET` list, `POST /items` add, `DELETE /items` remove, `PUT` replace
- **Sharing:** owner may grant `viewer` or `editor` access (`/api/v1/watchlist/shares`); viewers read-only; editors add/remove symbols only; audit at `/api/v1/watchlist/audit` — see `docs/features/watchlist-sharing.md`
- SQLite store (default `data/watchlist.db`, max 200 items/client); UI may also persist to `localStorage`
- Not suitable for multi-tenant production without real authentication
