# Spot order book

## Goal

Show a Binance-style **spot** order book for a pair (e.g. BTC/USDT): resting buy and sell size at each price, with **grouping** (0.01, 0.1, 1, …) and **wall** highlights. Grouping is computed on the **backend** so the frontend only renders ready rows. The same payload is available to the AI assistant.

Futures / other markets are out of scope for now version.

## Behavior

- `GET /api/v1/market/orderbook?exchange=binance&symbol=BTCUSDT&group=0.1&limit=20&rangePct=2`
  - `group` optional — omit for a suggested default from last/mid price
  - `limit` is **grouped rows per side** (5–100, default 20)
  - `rangePct` is the ±% of mid used for pressure/wall analysis (0.25–10, default **2**)
- Response includes:
  - `bids` (best first, highest price) and `asks` (best first, lowest price)
  - `quantity`, `notional`, running `cumulative` / `cumulativeNotional`
  - `isWall` when a bucket is unusually large vs the median on that side
  - `suggestedGroupSizes` for the step control
  - `spread`, `spreadPct`, `imbalance` (positive = more resting bids on the **visible ladder**)
  - `analysis` — buy/sell **pressure**, notional **imbalance**, and large **walls** from every
    live level within ±`rangePct` of mid (not only the first few orders). Nested `bands`
    (0.5 / 1 / 2 / 5%) show near vs farther depth. Same logic on Binance, Coinbase, and Bybit.
- `GET /api/v1/market/orderbook/combined?symbol=BTCUSDT&rangePct=2`
  - Sums live bid/ask **notional** from Binance + Coinbase + Bybit only in a
    **symmetric ±%** every venue can reach on both sides (the smaller of common
    bid reach and common ask reach). If all cover ±`rangePct` both ways, that
    requested band is used (`requestedReached`, `usedRangePct`). USD≈USDT.
  - Response: market-wide `pressure` / `imbalance`, per-venue breakdown, tagged walls.
- `GET /api/v1/market/orderbook/impact?symbol=BTCUSDT&quantity=5&side=buy`
  - Walks live resting depth like a market order. Buy consumes **asks** (lowest first);
    sell consumes **bids** (highest first).
  - Provide **quantity** (base, e.g. 5 BTC) **or** **notional** (quote, e.g. `1000000000`).
  - `exchange=all` (default) merges Binance + Coinbase + Bybit and takes the best prices first;
    pass `binance` / `coinbase` / `bybit` for one venue.
  - Returns `averagePrice`, `slippagePct` (vs mid), `slippageVsBestPct`, `endPrice` /
    `impactPct` (how far the last fill moved the book), `exhausted` if the visible book
    ran out, plus a short `fills` walk (first 30 levels).
  - Simulation only — not a quote, fill, or financial advice. Visible depth is often
    much thinner than a real $1B print; `exhausted=true` is the honest answer then.
- Grouping: bids floor to the step, asks ceil to the step (same idea as exchange UIs).
- All three venues keep a **live local book**. A dropped connection or missed update **invalidates** the copy and resyncs. Unsynced books are not served.
  - **Binance:** `@depth@100ms` + REST snapshot; `U`/`u` vs `lastUpdateId`.
  - **Coinbase:** public Exchange `level2_batch` snapshot + `l2update`, plus `heartbeat` timeout and periodic REST best-bid/ask price checksum.
  - **Bybit:** `orderbook.200.{symbol}` snapshot then `delta`; `u` must be contiguous (`u=1` overwrites as a venue restart snapshot).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/orderbook.go`, `orderbook_analysis.go`, `orderbook_combine.go`, `orderbook_impact.go`, `depthbook.go` |
| Adapters | `adapter/{binance,coinbase,bybit}/depthhub.go` |
| Service | `backend/internal/service/market` `GetSpotOrderBook`, `GetCombinedOrderBookAnalysis`, `EstimateOrderBookImpact` |
| HTTP | `GET /api/v1/market/orderbook`, `GET /api/v1/market/orderbook/combined`, `GET /api/v1/market/orderbook/impact` |
| MCP / AI | `get_spot_orderbook`, `analyze_spot_orderbook`, `analyze_market_orderbook`, `estimate_market_impact`, `create_orderbook_alert` |
| UI | `frontend` coin detail `OrderBookPanel` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/adapter/binance/ ./internal/adapter/coinbase/ ./internal/adapter/bybit/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/orderbook?symbol=BTCUSDT&group=0.1&rangePct=2"
curl "http://localhost:8080/api/v1/market/orderbook?exchange=coinbase&symbol=BTC-USD&rangePct=2"
curl "http://localhost:8080/api/v1/market/orderbook?exchange=bybit&symbol=BTCUSDT&rangePct=5"
curl "http://localhost:8080/api/v1/market/orderbook/combined?symbol=BTCUSDT&rangePct=2"
curl "http://localhost:8080/api/v1/market/orderbook/impact?symbol=BTCUSDT&quantity=5"
curl "http://localhost:8080/api/v1/market/orderbook/impact?symbol=BTCUSDT&notional=1000000000&exchange=all"
```

`live=true` and `source=websocket` when the local book is synced. Read `analysis.pressure`, `analysis.imbalance`, and `analysis.walls`. For impact, read `averagePrice`, `slippagePct`, `impactPct`, and `exhausted`.

## Limits / follow-ups

- Spot only.
- Idle streams are dropped after `ORDERBOOK_IDLE_TTL` (default 90s) and started again on the next request.
- Walls are a heuristic (size vs median / share of visible book), not exchange-labeled iceberg orders.
