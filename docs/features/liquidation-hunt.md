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
| `upScore` / `downScore` | 0–100 ease / likelihood for that direction, with `level`, `coverage`, `factors`, and `reasons` |
| `coverage` | How complete the inputs are (`complete` / `usable` / `thin` / `insufficient`); `usable=false` venues are shown but **not** mixed into combined `bias` |
| `bias` | `up` / `down` / `even` plus a one-line summary (venue and report-level). Combined lists `included` / `excluded` venues |

Zone bands and hunt P&L are **unchanged** by the scores. Scores only rank the two existing tours.

### Direction scores

Each direction is a weighted mix of data the desk already has:

| Factor | What it asks |
|---|---|
| Distance to zone | How far is the chosen target (`movePct`)? Closer is easier. |
| Spot walk cost | How much visible spot is needed vs the other side? Thinner push side helps. |
| Liq per spot | `efficiency` and whether the desk model is `profit` if cascade flow appears. |
| Price + OI trend | Same regime model as positioning (`long_buildup` / `short_covering` lean **up**; `short_buildup` / `long_unwinding` lean **down**). |
| Crowding + funding | Estimated long/short OI share and who pays funding. Shorts crowded + shorts paying favors **up**. |
| Taker + recent liqs | 1h aggressive buy/sell and 1h/4h observed liquidations. Buy-heavy / short-liq-heavy tape favors **up**. |

Missing or failed inputs are marked `missing` / `weak` / `error` on `coverage.inputs`. Time-based inputs (1h taker, 1h/4h liquidations, OI history, 1h/4h price) include `have` / `need` / `coverPct` — a few minutes of taker prints is **not** a complete hour. An OI (or other) sample older than the requested window is marked `stale` with `age` and is **not** used as that window's change; the trend score shrinks toward 50. Each `upScore` / `downScore` factor includes `sharePct` (mix weight) and `effect` (signed points versus 50) so you can see what moved the direction. Incomplete lookbacks shrink that factor's weight and pull the remaining score toward 50. Combined `bias` is **OI-weighted across usable venues only**. A venue that returns an error (book down, no OI, etc.) stays on the report for the user to see but is listed in `bias.excluded` and does **not** change the combined lean. One venue is never filled from the other.

`level` is `easier` (≥70) / `likely` (≥55) / `mixed` (≥40) / `hard`. Lean flips only when the two scores differ by at least 8 points.

Web: `/liquidations?view=hunt` — side-by-side up vs down with scores, target, spot cost, estimated liq, efficiency, desk result, and reasons.

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
| Domain | `backend/internal/domain/liquidation_hunt.go`, `liquidation_hunt_score.go`, `liquidation_hunt_heatmap.go`, `liquidation_hunt_heatmap_review.go`, `orderbook_reach.go` |
| Service | `backend/internal/service/market/hunt.go`, `hunt_heatmap.go` |
| HTTP | `GET /api/v1/market/liquidation-hunt`, `GET /api/v1/market/liquidation-hunt/heatmap` |
| MCP / AI | `estimate_liquidation_hunt`, `get_liquidation_heatmap` |
| Web | `/liquidations?view=hunt` — `frontend/src/components/organisms/LiquidationHunt`; `/liquidations?view=heatmap` — `LiquidationHeatmap` |

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
prices**. If Bybit candles are missing, the Bybit grid stays empty and
`missingVenues` includes `bybit`. `combined` sums a column **only when both
venues had their own price and OI** — a missing venue is not filled from the
other. Each grid has that venue's own `lastPrice` (combined omits it). `longs`
are longs that would liquidate if price falls into that bin; `shorts` are
shorts that would liquidate if price rises into that bin; `totals` = longs + shorts.

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

`signals[]` lists each hot area (time, price band, side) with a 1h / 4h / 12h
row: `hit` / `miss` / `pending` / `price_gap` / `liq_gap`, time-to-hit,
how far later candles reached (`priceCoveredSec` of `horizonSec`), and
observed liquidation notional before vs after. Gaps are labeled (`no_price`,
`price_short`, `no_liq`, `liq_short`) and are not filled from the other
exchange.

MCP: `get_liquidation_heatmap`.

On the product web app the same grid is the **Liquidation heatmap** on coin
detail → Tape: ranges 12h / 24h / 3d / 7d, venue Combined / Binance / Bybit,
and All / Longs / Shorts, plus the 1h / 4h / 12h hit-rate table. Combined is
the sum of venue cells.
