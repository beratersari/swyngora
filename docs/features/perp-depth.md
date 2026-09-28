# Perpetual depth (bid below vs ask above)

## Goal

Spot order books are already on the desk. For **BTC perpetual** (and other
linear USDT perps), show how much liquidity is waiting **below** last vs
**above** it — for example **40M bid within 1% below** and **15M ask
above**. Binance, Bybit, and OKX separately, plus combined.

## Behavior

`GET /api/v1/market/perp-depth?symbol=BTCUSDT`

- `exchange=all` (default): **Binance USD-M**, **Bybit linear**, and
  **OKX USDT-swap** separately plus `combined`. OKX size is contracts
  times `ctVal` (base), then notional is price times base.
- `rangePct` is the headline band (default **1**, max 5)
- `headline.below` is resting **bid** notional inside that % below mid
- `headline.above` is resting **ask** notional inside that % above mid
- `lean` is `bid` | `ask` | `balanced`
- `bands` also include 0.1 / 0.25 / 0.5 / 2 (and the requested %)
- `complete` is false when the snapshot does not reach that %
- Combined **adds** each venue's own ±% (not a shared price grid)

This is the **futures** book, not spot. Spot versus this perp book is
[`spot-perp-book.md`](spot-perp-book.md). Informational only — not
financial advice.

Web: coin detail **Flow** tab lists each venue's bid / ask under the combined total.

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
- `bidPct` / `askPct` are that mix as a percent of the book (bids 65% / asks 35%)
- `windows` (15m / 1h / 4h) compare that mix to a sample from then (bids
  55% → 65%, `changePct` +10) and `drivers` rank which coins pushed the
  change. The same windows compare total bid+ask (`pastTotal` → `total`,
  `totalChangePct`). A move of at least 5% is `depthKind` `thicker` or
  `thinner`; a smaller move is `steady`. `sizeDrivers` ranks the coins
  that added or removed the most of that size (ETH +100M ahead of BTC +20M).
  `history` is total bid+ask each minute for the last 6 hours, with BTC's
  close on the same minutes, so a slow
  thicken looks different from a jump. `moves` are the largest jumps and
  drops of at least 5% that finish within 15 minutes (up to three of each,
  no overlap). Each names BTC's close at the start, at the end, and about
  15 minutes later when that minute exists. A BTC move under 0.2% is flat.
  `drivers` rank the coins that added or removed the most of that move.
  `span` is `one` when one coin is at least half the move, `few` when two
  coins are at least 60%, and `broad` when the change is spread across the book.
  The Markets perp card lists them under the six-hour chart. Telegram `/perpliq`
  prints up to three of those sentences.
  `complete` is false until the
  desk has sampled that lookback.
- `bands` (0.1 / 0.25 / 0.5 / 1, up to `rangePct`) show how close that
  book is to mid. Example: **80M inside 0.1%** is **8% of the 1% book**;
  by 1% it is 1.0B. `leaders` are who fills that distance; `addedLeaders`
  are who added the extra size vs the previous tighter band.
- `exchange=all` (default) adds Binance USD-M, Bybit linear, and OKX USDT-swap per coin
- MCP: `get_perp_depth_overview`

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/perp_depth.go`, `perp_depth_overview.go` |
| Adapters | `adapter/binance/futures_depth.go`, `adapter/bybit/futures_depth.go`, `adapter/okx/futures_depth.go`, `adapter/perpdepthstore` |
| Service | `GetPerpDepth`, `GetPerpDepthOverview`, `PerpDepthHistoryWorker` |
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
