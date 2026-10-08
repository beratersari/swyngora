# Cumulative volume delta (CVD)

## Goal

See the **buy vs sell difference over time**, not just the last few
minutes. CVD keeps adding aggressive **market-buy** notional minus
**market-sell** notional, then plots that path next to **price**.

Useful when:

- price barely moves but there are many market buys (absorption)
- price goes up but the sell side is actually stronger (divergence)

## Behavior

`GET /api/v1/market/cvd?symbol=BTCUSDT&range=7d`

- `exchange=all` (default): **Binance**, **Bybit**, and **OKX** separately plus `combined`
- `range`: `24h` (default, 5-minute bars), `7d` (30-minute bars), `30d` (2-hour bars), `90d` (4-hour bars). Longer ranges fold the stored minutes. The series is `complete` only when those minutes reach the start of the range
- Binance minutes for the liquid USDT list (up to 100 names, seeds first) are filled in the background from public USD-M aggregate trades, about 7 days, one hour at a time. A rate-limit pause retries that window and does not store a partial hour. Bybit and OKX are not backfilled this way
- Points: `buyNotional`, `sellNotional`, `delta`, running `cvd`,
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
  The coin price chart draws futures and spot separately. Each market
  gets a line since 00:00 UTC and a line since the current 4-hour open,
  labeled with Futures or Spot and the venue (or Combined). On a range
  longer than 24h it also draws that market's running sum. Buy−sell
  `delta` stays the futures histogram. Combined is drawn only when every
  expected venue's own tape is complete and reaches the page venue.
  Futures expects Binance, Bybit, and OKX. Spot expects Binance and Bybit.
  Otherwise the chart draws the page venue and names who is missing.
  A Coinbase or OKX page uses Binance for spot and says Binance.
- Combined **adds** each venue’s delta on the overlapping **time range**
  (a quiet 5m slot on one side is treated as 0, not a hole). Combined
  `complete` is false when any venue passed in has no points or has not
  reached the window open. One missing 5m bucket at the start, when every
  venue otherwise covers the lookback, does not make a 24h tape look
  unfinished. A short Bybit history still does **not** make combined
  complete. `overlapFrom` / `overlapTo` mark that range.
- Combined points and `contributions` show how much **Binance** and
  **Bybit** each added. `venueSplit` flags when one venue’s CVD rose
  while the other fell (`alignment=opposite`)
- **Spot vs futures:** `venues` / `combined` are **futures**; `spotVenues`
  / `spotCombined` are **spot**. `spotFutures` flags when spot CVD and
  futures CVD go different ways (spot buying while perps sell, or the
  reverse) on 15m / 1h / 4h / 24h. Spot Binance uses 5m kline taker-buy
  volume (~24h immediately); Bybit spot uses live trades (fills in over
  time). Works for BTC and any other USDT pair.
- A divergence **run** is only consecutive bars of the same kind. If
  it stops and the same split appears later, that is a new episode.
  The coin chart marks that current run once, at its start, on the same
  venue as the CVD line (combined only when combined is complete and
  reaches as far as the page venue; otherwise the page venue, and a
  venue that starts later is named). All and Order flow show the mark
  while CVD is on. Positioning, Liquidity, and equities do not. Agreement
  draws nothing. A later bar where price and CVD move the same way ends the
  run; flat bars after it do not. A 4-hour window that disagrees without
  `since` is not a run.
- The derivatives board column `cvdSplit` is that same current run for the
  liquid USDT list. It stays blank until the tape reaches 00:00 UTC, the
  newest minute is fresh, and the newest bar has a price. `agrees` means
  the tape is definite and there is no current run. Combined stays blank
  unless every expected venue's 24-hour series is complete. See
  [`derivatives-board.md`](derivatives-board.md).

Futures CVD on Binance, Bybit, and OKX is the sum of measured prints
(price × size). Minutes are stored in `data/futures.db` (`taker_buckets`)
while the server is recording. A 24h series is `complete` only after
that tape covers it. A 7d, 30d, or 90d series uses the same minutes and
is `complete` only when the oldest stored minute reaches the start of
that range. The coin chart keeps the day and 4-hour anchors for futures
and for spot, and on a longer range also draws each market's running sum.
Older
Binance rows that were base volume × the mark were dropped once (schema
version 6) so they are not added on top of the measured minutes.

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
curl "http://localhost:8080/api/v1/market/cvd?symbol=BTCUSDT&range=7d"
```
