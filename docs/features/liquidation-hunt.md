# Hypothetical liquidation hunt

## Goal

Answer a **what-if** only: if a desk walked **this venue’s spot book** to push
price and force futures liquidations, where is the main long/short pressure,
how much visible spot buying or selling that takes, and would the tour
approximately make or lose money?

This is **not** evidence that Binance or Bybit do this. It is not financial
advice.

## Behavior

`GET /api/v1/market/liquidation-hunt?symbol=BTCUSDT`

- `exchange=all` (default) returns **Binance USD-M and Bybit linear separately**.
  They are never averaged. `exchange=binance` or `bybit` for one venue.
- **Up hunt:** buy spot → lift price → estimated shorts liquidate → they must
  buy to cover. Then unwind the spot (old bids vs assumed cascade exit at the
  target).
- **Down hunt:** sell spot → push price down → estimated longs liquidate →
  buy back cheaper if cascade flow appears.

Each venue report includes:

| Field | Meaning |
|---|---|
| `upPressure` / `downPressure` | Estimated liquidation bands (leverage mix × blended OI share) plus 24h observed clusters |
| `upHunt` / `downHunt` | Primary target (best estimated liq vs spot cost), spot walk, and P&L |
| `spot.notional` | Visible spot size to walk last price to the target |
| `bookOnlyPnl` | Unwind the same size on the **current** opposite side (usually a loss) |
| `netWithCascade` | Part of estimated liquidations becomes exit flow at the target |
| `houseEdge` | `profit` / `loss` / `unreachable` from `netWithCascade` |
| `efficiency` | estimated liquidated notional ÷ spot notional |

### Assumptions (also returned on `assumptions`)

- Isolated liquidation distance ≈ `1/leverage − 0.4%` maintenance, entries at
  the current price.
- Default leverage mix: 5× 15%, 10× 25%, 25× 30%, 50× 18%, 75× 7%, 100× 4%,
  125× 1%. Shifted toward high leverage on the funding-paying side.
- Published long/short is **account count**. Position-share proxy is blended
  60% toward 50/50.
- `liquidationTake` = 0.5% of estimated liquidated notional (insurance-fund-like
  stand-in, not a published fee).
- Spot taker 0.10% each way. Cascade fill rate 50% of estimated liq notional.
- Mark price is a **multi-venue index**. Walking one spot book may not move
  mark 1:1. Hidden liquidity is ignored.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation_hunt.go`, `liquidation_hunt_heatmap.go`, `liquidation_hunt_heatmap_review.go`, `orderbook_reach.go` |
| Service | `backend/internal/service/market/hunt.go`, `hunt_heatmap.go` |
| HTTP | `GET /api/v1/market/liquidation-hunt`, `GET /api/v1/market/liquidation-hunt/heatmap` |
| MCP / AI | `estimate_liquidation_hunt`, `get_liquidation_heatmap` |
| Web | Coin detail **Tape** tab — `frontend/src/components/organisms/LiquidationHeatmap` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1
curl "http://localhost:8080/api/v1/market/liquidation-hunt?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/liquidation-hunt/heatmap?symbol=BTCUSDT&range=24h"
```

Read `venues[].upHunt` and `venues[].downHunt`. Treat `houseEdge` as a model
output, not a claim about any exchange.

## Price × time heatmap

`GET /api/v1/market/liquidation-hunt/heatmap?symbol=BTCUSDT&range=24h`

CoinGlass-style grid: **time on X**, **price on Y** (highest bin first), **color =
estimated liquidation notional** at that price in that column.

| `range` | Window | Column step |
|---------|--------|-------------|
| `12h` | 12 hours | 15 minutes |
| `24h` | 24 hours | 30 minutes |
| `3d` | 3 days | 1 hour |
| `7d` | 7 days | 2 hours |

Each column uses historical **open interest**, **that venue's own price**
(spot candles from that exchange only), the same **leverage mix** as the hunt
(tilted by funding), and blended **account long/short**. Observed liquidation
prints in that column are added into the matching price bin.

`binance` and `bybit` are modeled separately and **never borrow each other's
prices**. If Bybit candles are missing, the Bybit grid stays empty. `combined`
is the **sum** of their cells (not an average). `longs` are longs that would
liquidate if price falls into that bin; `shorts` are shorts that would
liquidate if price rises into that bin; `totals` = longs + shorts.

### Did the hot zones work? (`review`)

After the grid is built, each **high** area (contiguous bins at ≥ 60% of that
column's peak, excluding the bin that already holds the venue mark) is checked
against **later candles from the same venue**:

| Horizon | Question |
|---------|----------|
| `1h` / `4h` / `12h` | Did that venue's own price trade through the zone? How long did it take? Did observed liquidations in that zone rise vs the same-length window before the signal? |

- **Validated** — that venue's later candles span the horizon **and**
  liquidation history covers the same-length windows before and after.
- **Missing** — too recent (`pending`), no venue-local forward price path
  (`priceMissing`), or no liquidation history covering the window
  (`liqMissing`). Not filled from the other exchange.
- **Hit / false / hit rate** — scored only on **validated** signals.
- **Liqs rose** — among validated hits, whether observed notional in the zone
  rose vs the prior window.
- **Combined** uses the summed grid and counts a hit if **either** venue's own
  price reached the zone.

MCP: `get_liquidation_heatmap`.

On the product web app the same grid is the **Liquidation heatmap** on coin
detail → Tape: ranges 12h / 24h / 3d / 7d, venue Combined / Binance / Bybit,
and All / Longs / Shorts, plus the 1h / 4h / 12h hit-rate table. Combined is
the sum of venue cells.
