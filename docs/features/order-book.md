# Spot order book

## Goal

Show a Binance-style **spot** order book for a pair (e.g. BTC/USDT): resting buy and sell size at each price, with **grouping** (0.01, 0.1, 1, …) and **wall** highlights. Grouping is computed on the **backend** so the frontend only renders ready rows. The same payload is available to the AI assistant.

Futures / other markets are out of scope for now version.

## Behavior

- `GET /api/v1/market/orderbook?exchange=binance&symbol=BTCUSDT&group=0.1&limit=20`
  - `group` optional — omit for a suggested default from last/mid price
  - `limit` is **grouped rows per side** (5–100, default 20)
- Response includes:
  - `bids` (best first, highest price) and `asks` (best first, lowest price)
  - `quantity`, `notional`, running `cumulative` / `cumulativeNotional`
  - `isWall` when a bucket is unusually large vs the median on that side
  - `suggestedGroupSizes` for the step control
  - `spread`, `spreadPct`, `imbalance` (positive = more resting bids)
- Grouping: bids floor to the step, asks ceil to the step (same idea as exchange UIs).
- Short TTL cache (~2s, `ORDERBOOK_CACHE_TTL`) + singleflight per venue.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/orderbook.go` |
| Adapters | `backend/internal/adapter/{binance,coinbase,bybit}/orderbook.go` |
| Service | `backend/internal/service/market` `GetSpotOrderBook` |
| HTTP | `GET /api/v1/market/orderbook` |
| MCP / AI | `get_spot_orderbook` |
| UI | `frontend` coin detail `OrderBookPanel` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/adapter/binance/ ./internal/adapter/coinbase/ ./internal/adapter/bybit/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/orderbook?symbol=BTCUSDT&group=0.1"
```

Open `/markets/binance/BTCUSDT` and change the group steps next to the chart.

## Limits / follow-ups

- Spot only. No futures or live WebSocket depth stream yet (HTTP poll).
- Walls are a heuristic (size vs median / share of visible book), not exchange-labeled iceberg orders.
- Coinbase level-2 is already price-aggregated by the venue (top ~50); we still group on top.
