# Perpetual depth (bid below vs ask above)

## Goal

Spot order books are already on the desk. For **BTC perpetual** (and other
linear USDT perps), show how much liquidity is waiting **below** last vs
**above** it — for example **40M bid within 1% below** and **15M ask
above**. Binance and Bybit separately, plus combined.

## Behavior

`GET /api/v1/market/perp-depth?symbol=BTCUSDT`

- `exchange=all` (default): **Binance USD-M** and **Bybit linear**
  separately plus `combined`
- `rangePct` is the headline band (default **1**, max 5)
- `headline.below` is resting **bid** notional inside that % below mid
- `headline.above` is resting **ask** notional inside that % above mid
- `lean` is `bid` | `ask` | `balanced`
- `bands` also include 0.1 / 0.25 / 0.5 / 2 (and the requested %)
- `complete` is false when the snapshot does not reach that %
- Combined **adds** each venue's own ±% (not a shared price grid)

This is the **futures** book, not spot. Informational only — not
financial advice.

## Market-wide total

`GET /api/v1/market/perp-depth/overview`

Same idea as breadth: look at the **liquid coins we already follow**
(top USDT perps, default **20**, max 40; leveraged tokens and stables
like USDC are dropped) and add up how much bid is waiting within 1%
below each coin's mid versus how much ask is waiting above. Then rank
which coins make up the biggest part of that book (example: **BTC 48% /
ETH 17%** of 2.4B bid + 1.8B ask).

- `bid` / `ask` / `total` are USDT notional
- `coins[].sharePct` is that coin's percent of bid+ask
- `coins[].bidSharePct` / `askSharePct` are percent of each side
- `leaders` are the top bases by total share
- `exchange=all` (default) adds Binance USD-M and Bybit linear per coin
- MCP: `get_perp_depth_overview`

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/perp_depth.go`, `perp_depth_overview.go` |
| Adapters | `adapter/binance/futures_depth.go`, `adapter/bybit/futures_depth.go` |
| Service | `GetPerpDepth`, `GetPerpDepthOverview` |
| HTTP | `GET /api/v1/market/perp-depth`, `GET /api/v1/market/perp-depth/overview` |
| MCP / AI | `get_perp_depth`, `get_perp_depth_overview` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/... -run PerpDepth
curl "http://localhost:8080/api/v1/market/perp-depth?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/perp-depth?symbol=BTCUSDT&exchange=binance&rangePct=1"
curl "http://localhost:8080/api/v1/market/perp-depth/overview"
curl "http://localhost:8080/api/v1/market/perp-depth/overview?limit=20&rangePct=1"
```
