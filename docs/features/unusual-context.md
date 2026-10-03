# Unusual versus stored history

## Problem / goal

A funding print of 0.03% or a one-hour liquidation total is hard to read on its own. This read says whether that figure is high, typical, or low versus the same coin’s own stored samples on that venue.

## Behavior

`GET /api/v1/market/unusual?symbol=BTCUSDT&exchange=binance|bybit|okx|all`

`exchange=all` (the default) returns Binance, Bybit, and OKX as separate blocks. There is no combined percentile.

Each block has six readings, in this order:

| Metric | What is scored | Past samples | Minimum |
|---|---|---|---|
| Funding | Fresh predicted rate when one was stored in the last 30 minutes, otherwise the latest settlement | Stored settlements only | 20 |
| Open interest | Latest non-overlapping 24h percent change | Earlier 24h changes from hourly stored open interest | 20 |
| Liquidations | Last completed clock hour of notional | Earlier completed hours. Quiet hours inside the first-to-last stored print count as zero | 24 |
| CVD | Last completed UTC day of aggressive buy minus sell | Earlier stored days. A missing day is left out | 20 |
| Spread | Latest hour’s average top-of-book spread percent | Earlier hours from the spot book, kept about 30 days | 24 |
| Print size | Always unknown | Only the previous 15 minutes exist | — |

The current point is not part of the past sample. The percentile is the midrank of the current value in that past, rounded to the nearest integer.

| Percentile | Band |
|---|---|
| 98 and above | `extreme_high` |
| 90–97 | `high` |
| 11–89 | `typical` |
| 3–10 | `low` |
| 2 and below | `extreme_low` |

Fewer samples than the minimum stay `unknown` with reason `few_samples`. No usable sample stays `unknown` with reason `no_sample`. OKX has no spot book here, so its spread is `no_book` and the book is not queried. Print size is `no_stored_history`.

`spanHours` is the first-to-last span of the past samples actually stored. It is not a promised 30 or 90 days. The English `summary` on each reading is for the assistant. The coin page builds its own sentence from English and Turkish copy.

The coin Tape tab shows this card above the rest of the tape. Informational only — not financial advice.

## Market scan

`GET /api/v1/market/unusual/scan?exchange=binance|bybit|okx|all&metric=all|funding|open_interest|liquidations|cvd|spread&limit=100`

The scan scores the liquid USDT list the recorder keeps: the seed pairs first, then the highest-volume names, capped at 100. The page opens on 36. It uses the same per-venue rules as the coin read. A row is kept only when the band is high, low, extremely high, or extremely low. Typical readings and readings with too few samples are left off. `quiet` is how many scanned coins had no such row. `distance` is how far the percentile sits from 50, and the list is ranked by that. Venues stay separate. Print size is not a scan metric.

The page is `/unusual`. A row opens that coin's Tape tab. An OKX row opens the Binance tape, because OKX is not a spot page. The response is cached for about 45 seconds. Informational only — not financial advice.

The shorter “versus typical” reads stay as they are: spread widen on the order book, liquidation cascade bursts, volume surge, options moves, and print size versus the prior 15 minutes.

## Honesty limits

- Open interest and liquidations are hourly SQL aggregates. They are not scored from the newest 2,500 snapshot or liquidation rows, because 5-minute open-interest samples would crowd out the older hours.
- Spread is the hourly average of stored `spread_pct`. Ladder payloads are not loaded.
- The read does not call the live exchange.
- Binance open interest in the archive is only the latest 30 days that endpoint documents. The card reports the span those rows cover.
- Public liquidation refill is about 7 days. Older hours exist only when this process already recorded them. Quiet hours are filled only between the first and last stored print.
- CVD days with no stored bars are not treated as zero.
- Long/short is not part of this read.

## Where the code lives

| Layer | Path |
|---|---|
| Scorer | `backend/internal/domain/unusual.go` |
| Futures SQL | `backend/internal/adapter/futuresstore/unusual.go` |
| Book SQL | `backend/internal/adapter/bookhiststore/unusual.go` |
| Service | `backend/internal/service/market/unusual.go` |
| HTTP | `GET /api/v1/market/unusual` and `GET /api/v1/market/unusual/scan` |
| MCP / AI | `get_unusual_context`, `scan_unusual` |
| Tape card | `frontend/src/components/organisms/UnusualContext/` |
| Market page | `frontend/src/components/pages/UnusualPage` |

## How to test

```bash
cd backend && go test ./internal/domain/ -count=1 -run Unusual
cd backend && go test ./internal/adapter/futuresstore/ -count=1 -run Unusual
cd backend && go test ./internal/adapter/bookhiststore/ -count=1 -run HourlySpread
cd backend && go test ./internal/service/market/ -count=1 -run Unusual
cd backend && go test ./internal/transport/http/handler/ -count=1 -run GetUnusual
cd frontend && npx vitest run src/components/organisms/UnusualContext src/components/organisms/UnusualScanBoard src/components/pages/UnusualPage
```

## Known limitations

- Print size cannot be called usual or unusual until a longer series is stored.
- A venue that has been recorded for only a few days stays unknown.
- OKX spread stays unknown because this product does not keep an OKX spot book.
- OKX CVD appears only when taker bars were stored for that venue.
- The market scan does not invent a combined percentile. One coin can appear once per venue and series.
- Coins outside the liquid 100 are not on `/unusual`. Open the coin tape for those.
