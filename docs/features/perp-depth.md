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

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/perp_depth.go` |
| Adapters | `adapter/binance/futures_depth.go`, `adapter/bybit/futures_depth.go` |
| Service | `GetPerpDepth` |
| HTTP | `GET /api/v1/market/perp-depth` |
| MCP / AI | `get_perp_depth` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/... -run PerpDepth
curl "http://localhost:8080/api/v1/market/perp-depth?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/perp-depth?symbol=BTCUSDT&exchange=binance&rangePct=1"
```
