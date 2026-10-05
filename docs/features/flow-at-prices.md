# Flow at prices (buy / sell / delta per price)

## Goal

See **what printed at each price**, not only how much bought and sold
inside a candle. If BTC traded 65400–65600, the candle totals cannot
say how much buy vs sell happened at **65520**, how many aggressive
prints hit that row, or what the **delta** was.

## Behavior

`GET /api/v1/market/flow-at-prices?symbol=BTCUSDT`

- Works for BTC and other linear USDT pairs the venues list
- `exchange=all` (default): **Binance** USD-M and **Bybit** linear
  separately plus `combined`
- Time range: `window=5m|15m|1h|4h|1d|3d|7d|30d` (default `15m`) **or**
  `startTime` + `endTime` (RFC3339 or unix ms). Max **30 days**
- Optional `tickSize` (price row width, e.g. `10`). Omit to pick a row
  size that keeps the ladder readable (widened if it would exceed 200
  rows)
- Optional `minPrice` / `maxPrice` clip the ladder to a band (e.g.
  `65400`–`65600`)
- Each level: `price`–`high`, `buyVolume` / `sellVolume` (USDT),
  `buyCount` / `sellCount` / `trades`, `delta` (buy − sell), `buyPct`,
  share of total, `dominant`, `isPoc`
- Levels are **highest price first**
- **POC:** the row with the most aggressive volume (ties go to the row
  closer to last)
- `truncated` / `complete`: the ladder may not cover the whole window.
  The live tape is the Binance, Bybit, and OKX trade stream once it
  reaches the start of the window; until then the newest public trades
  are used. Closed minutes are stored while the server runs (about
  30 days, `FLOW_AT_PRICES_RETENTION`) for the liquid recorder list
  plus pairs someone has opened. Binance minutes for that liquid list
  are also filled from public USD-M aggregate trades, about 7 days,
  one hour at a time. Bybit and OKX minutes before the process started
  are still missing. A 4h / 1d / 3d / 7d / 30d lookback, and
  Bybit further back than those 1000 trades, reads that stored buy,
  sell, delta, and print count. Stored rows are about 0.05% of price wide and roll up to a
  wider `tickSize`

This is **not** volume profile. Volume profile spreads each candle's
volume evenly across high–low. Flow at prices bins **actual aggressive
prints** onto the prices they traded.

`trades` is aggregated print count (Binance `aggTrades` can merge
fills), not individual order IDs.

Informational only — not financial advice.

## Candle footprint

`GET /api/v1/market/flow-at-prices/candles?symbol=BTCUSDT&interval=1h`

The coin chart uses this read. It splits the same stored minutes into
the candle they traded in, then into price rows inside that candle
(at most 24). `interval` matches the chart (`1m` through `1M`).
`startTime` and `endTime` are optional and must be set together.
A missing range is the newest 120 candles. The read keeps at most
400 candles and 30 days, and sets `truncated` when it shortens the range.

Closed minutes come from the stored ladder. The current minute comes
from the live tape when that tape already covers it. This route does
not walk public trade history.

`exchange=all` returns each venue plus `combined`. `combined` is
present only when every configured venue was recorded for the range.
`missing` names a venue with no stored minutes. A missing venue is not
treated as zero. `isPoc` is the busiest row in that candle. `imbalance`
is `buy` or `sell` when that side is at least 3× the other and the row
is at least 1% of the candle.

The chart draws `combined` when it is present. Otherwise it draws the
page's venue and names the missing venues. When no stored minutes fill
the candles, the chart keeps the single right-edge ladder from
`/flow-at-prices`.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/flow_at_prices.go`, `flow_at_prices_history.go`, `candle_footprint.go` |
| Store | `backend/internal/adapter/flowstore` (`data/flowprices.db`) |
| Service | `backend/internal/service/market/flowatprices.go`, `candlefootprint.go` |
| HTTP | `GET /api/v1/market/flow-at-prices` and `GET /api/v1/market/flow-at-prices/candles` |
| MCP / AI | `get_flow_at_prices`, `get_candle_footprint` |
| Web | Coin detail **Flow** tab (`FlowFootprint`): sell left, buy right, delta, POC highlight, and a badge when one side is at least 3× the other. The Overview chart paints the same buy and sell inside each candle |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run "FlowAtPrices|CandleFootprint"
curl "http://localhost:8080/api/v1/market/flow-at-prices?symbol=BTCUSDT&window=15m"
curl "http://localhost:8080/api/v1/market/flow-at-prices?symbol=BTCUSDT&minPrice=65400&maxPrice=65600&tickSize=10"
curl "http://localhost:8080/api/v1/market/flow-at-prices/candles?symbol=BTCUSDT&interval=1h"
```
