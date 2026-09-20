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
- Time range: `window=5m|15m|1h` (default `15m`) **or** `startTime` +
  `endTime` (RFC3339 or unix ms). Max **1 hour**
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
- `truncated` / `complete`: the venue tape may not cover the whole
  window (Binance pages aggTrades; Bybit is the newest ~1000 public
  trades)

This is **not** volume profile. Volume profile spreads each candle's
volume evenly across high–low. Flow at prices bins **actual aggressive
prints** onto the prices they traded.

`trades` is aggregated print count (Binance `aggTrades` can merge
fills), not individual order IDs.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/flow_at_prices.go` |
| Service | `backend/internal/service/market/flowatprices.go` |
| HTTP | `GET /api/v1/market/flow-at-prices` |
| MCP / AI | `get_flow_at_prices` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run FlowAtPrices
curl "http://localhost:8080/api/v1/market/flow-at-prices?symbol=BTCUSDT&window=15m"
curl "http://localhost:8080/api/v1/market/flow-at-prices?symbol=BTCUSDT&minPrice=65400&maxPrice=65600&tickSize=10"
```
