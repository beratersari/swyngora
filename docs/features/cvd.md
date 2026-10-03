# Cumulative volume delta (CVD)

## Goal

See the **buy vs sell difference over time**, not just the last few
minutes. CVD keeps adding aggressive **market-buy** notional minus
**market-sell** notional, then plots that path next to **price**.

Useful when:

- price barely moves but there are many market buys (absorption)
- price goes up but the sell side is actually stronger (divergence)

## Behavior

`GET /api/v1/market/cvd?symbol=BTCUSDT`

- `exchange=all` (default): **Binance**, **Bybit**, and **OKX** separately plus `combined`
- 5-minute points: `buyNotional`, `sellNotional`, `delta`, running `cvd`,
  `price`, `priceChangePct`, `vsPrice`, and `divergence`
- **Divergence:** `price_up_cvd_down` or `price_down_cvd_up` on a point,
  on each 15m / 1h / 4h / 24h window, and on the series `divergence`
  object — how long the current split has been running (`duration`,
  `since`), and how far price (`priceMovePct`) and CVD (`cvdMove`)
  moved against each other
- Windows **15m / 1h / 4h / 24h**: CVD change vs price change (not only
  the latest total), labeled `confirms` | `opposite` | `absorption` | `mixed`
- **Anchors** reset the sum at a clock. `day` is 00:00 UTC. `4h` is the
  current 4-hour open (00, 04, 08, 12, 16, 20 UTC). At 10:30 the rolling
  4h window still starts at 06:30; the 4h anchor starts at 08:00. Each
  anchor reports `cvd` (buy minus sell since that time), price change,
  and the same confirms / opposite / absorption label. `complete` is
  false when the stored bars start after the anchor — the number is
  then only the bars on hand. The coin tape shows both next to last CVD.
  The coin price chart draws the same resets as two lines under the
  candles, plus each candle's buy−sell `delta` as a histogram. Both use
  the combined futures points when `exchange=all`.
- Combined **adds** each venue’s delta on the overlapping **time range**
  (a quiet 5m slot on one side is treated as 0, not a hole). Combined
  `complete` follows both venues — one missing 5m bucket at the start
  does not make a 24h tape look unfinished. A short Bybit history still
  does **not** make combined complete. `overlapFrom` / `overlapTo` mark
  that range.
- Combined points and `contributions` show how much **Binance** and
  **Bybit** each added. `venueSplit` flags when one venue’s CVD rose
  while the other fell (`alignment=opposite`)
- **Spot vs futures:** `venues` / `combined` are **futures**; `spotVenues`
  / `spotCombined` are **spot**. `spotFutures` flags when spot CVD and
  futures CVD go different ways (spot buying while perps sell, or the
  reverse) on 15m / 1h / 4h / 24h. Spot Binance uses 5m kline taker-buy
  volume (~24h immediately); Bybit spot uses live trades (fills in over
  time). Works for BTC and any other USDT pair.
- A divergence **run** is only consecutive 5m bars of the same kind. If
  it stops and the same split appears later, that is a new episode.

Futures CVD on Binance, Bybit, and OKX is the sum of measured prints
(price × size). Minutes are stored in `data/futures.db` (`taker_buckets`)
while the server is recording, and a 24h window is `complete` only after
that tape covers it. Older Binance rows that were base volume × the mark
were dropped once (schema version 6) so they are not added on top of the
measured minutes.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/cvd.go` |
| Adapters | `adapter/binance/tradehub.go`, `adapter/bybit/tradehub.go`, `adapter/okx/tradehub.go`, `adapter/futuresstore` |
| Service | `backend/internal/service/market/cvd.go` |
| HTTP | `GET /api/v1/market/cvd` |
| MCP / AI | `get_cvd` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/adapter/futuresstore/ -run CVD
curl "http://localhost:8080/api/v1/market/cvd?symbol=BTCUSDT"
```
