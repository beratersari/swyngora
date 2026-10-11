# Futures liquidations

## Goal

Show Coinglass-style **long vs short liquidation** totals for a coin over the last
**5 minutes, 1 hour, 4 hours, and 24 hours**, so a user or the AI can ask
“how much BTC was liquidated in the last 24 hours?”

## Behavior

- The coin price chart draws the strongest estimated **10× / 25× / 50× / 100×** long level at or below last and short level at or above last as dashed prices. Longs use the down color, shorts the up color. The full price × time map opens `/liquidations`.
- The liquidation heatmap ranges are **12h, 24h, 3d, 7d, 14d, 30d, and 90d**. On startup the futures archive pages public **1-hour** open interest, funding, and long/short for the seed pairs (BTC, ETH, SOL, XRP, BNB, DOGE, ADA, AVAX, LINK, SUI) into `data/futures.db`, each series only as far as that endpoint documents (Binance open interest and long/short: 30 days; OKX hourly open interest: 1,440 points; Bybit open interest back to listing, kept here for 90 days). See [`futures-history.md`](futures-history.md). 90-day columns are one day wide. A Binance column older than the public 30 days is only hours this process already recorded. Combined sums a column only when Binance, Bybit, and OKX all have their own price and open interest there. OKX prices are that venue's USDT-swap candles. The opening view of `/liquidations` is the measured print map. Model on that page is the leverage estimate only; force-order prints are not added into those cells. The level chart reaches the 5x distance. 5x is drawn in the 10x color, and 75x and 125x in the 100x color, each at its own distance. OKX hunt and max pain use the USDT-swap mark. Observed liquidation prints are kept for the futures retention. Public REST refills about 7 days, and a page that does not reach the start of a gap leaves that earlier time empty. The 24-hour reconnect fill is unchanged. Longer model columns are the leverage model on archived open interest. Windows past a day read one stored open-interest sample per step, and a column with no sample stays empty. The level chart range (12h through 30d) is the later-print review; the bars stay the live map. Binance calls stay behind the shared REST gate.
- `GET /api/v1/market/liquidations?symbol=BTCUSDT`
  - `exchange=all` (default) sums **Binance USD-M**, **Bybit linear perpetual**, and **OKX USDT-swap**
  - `exchange=binance`, `bybit`, or `okx` for one venue
- `GET /api/v1/market/liquidations/overview`
  - Market-wide **1h / 4h / 12h / 24h** totals from stored prints (no symbol)
  - `window` (default `24h`) ranks `coins` by total notional for a treemap
  - `limit` default 50, max 100
  - `coverage` names each venue fresh, old, or missing. `definite` is false when any venue is missing or old. The page still shows the stored totals and names that venue; it does not call the partial sum the whole market (`docs/features/venue-coverage.md`)
- Each window includes:
  - `longNotional` / `shortNotional` / `totalNotional` (USDT)
  - `count`
  - `biggest` (side, price, qty, notional, venue, time)
  - `complete` — false until **this coin on this venue** has had a **live websocket**
    for the full window. Time does not count if the socket never connects or drops.
    Combined `all` uses the shorter live coverage **and requires every futures
    venue** (Binance, Bybit, OKX). A venue that never started or is down is
    listed in `feed.missing` and is **not** replaced by another exchange.
- Streams (background, always on):
  - Binance: `wss://fstream.binance.com/market/ws/!forceOrder@arr` (USD-M symbols). At most the **largest** liquidation per symbol per ~1s. Coin-margined events on that merged stream (`st` 2) are left out.
  - Bybit: `wss://stream.bybit.com/v5/public/linear` topic `allLiquidation.{symbol}`. Seed majors + top linear USDT by 24h turnover; querying a symbol also subscribes it.
  - OKX: `wss://ws.okx.com:8443/ws/v5/public` channel `liquidation-orders` `instType=SWAP` (all USDT-swap). Contract size (`ctVal`) converts `sz` to base quantity. A missing or non-positive `ctVal` drops that print instead of storing it as 1 contract. The history window for that symbol stays uncovered until a later pass has the size. Prints already stored stay as they were written.
- `feed` on coin, overview, and chart responses:
  - per venue `live`, `lastEventAt` (last print), `lastSeenAt` (last websocket
    payload), `coverageSeconds`, `missingSeconds`, and `gaps` (still-unfilled
    disconnects in the last 6 hours)
  - `missing` lists venues that are down, stalled (>2 minutes silent), or never started
  - a restart inserts a downtime gap from the last coverage save so the book
    does not pretend it was live while the process was off
  - **Reconnect backfill:** when a venue socket comes back, that venue only
    tries its own historical liquidation API for each closed hole. Live prints
    and history that share venue + symbol + side + time + price + qty count
    once. If the source covers the whole hole, the gap is removed and that
    time is added to coverage. If only part of the window is confirmed, the
    leftover piece stays and `missingSeconds` is how much is still missing.
    No history (or a 404) leaves the gap as-is. Binance is never used to fill
    Bybit, and the other way around.
- Alerts (same `POST /api/v1/alerts` desk): `kind=liquidation_feed` fires once when
  Binance or Bybit stay down/stalled longer than `targetPrice` seconds (default 300)
  and re-arms only after the feed is live again. `kind=liquidation_cascade` fires
  once when a coin (or `symbol=all`) hits grade `cascade` or hotter; the webhook
  names the **exchange** and **coin**. `kind=liquidation_notional` fires when
  long, short, or total USDT in 1m/5m/15m/1h crosses a dollar line (one fire
  per wave). Creating a coin alert subscribes that pair and fills the lookback
  from history. See [`price-alerts.md`](price-alerts.md).
- Side meaning: **long** = long positions were force-closed; **short** = shorts were force-closed.
- The overview (`/liquidations`) and the all-coin chart read **stored** force-order prints in `futures.db`, then add prints newer than the newest saved row for that venue. The in-memory book is still 24 hours and is what the per-coin 5m/1h/4h/24h route uses. Every print is written to SQLite. On startup the last **24 hours** are reloaded into that book, including live-coverage clocks. A dropped persist queue is written synchronously instead of discarded. Stored rows: `GET /api/v1/market/futures-history?metric=liquidations`. See [`futures-history.md`](futures-history.md).
- Chart (`GET /liquidation-levels`): CoinGlass-style **horizontal** price bars
  from each venue's own last price, open interest, and a **10x / 25x / 50x / 100x**
  mix. Each bar includes the mass at that price (`byLeverage`) plus the walk
  from last to that level (`cumLong` / `cumShort` / `cumTotal` / `cumByLeverage`).
  Combined aligns venues by **percent from their own last** onto an
  **OI-weighted reference last** (`lastPrice`); `lastPrices` keeps each venue
  and a missing venue is never copied. `review` rebuilds those past maps from
  stored open interest and last price, then scores later **same-venue**
  liquidations in each 10x / 25x / 50x / 100x band (1h / 4h / 12h). Each
  sample keeps a **frozen map** at that last so shorts stay above it (open
  a past estimate on the chart — today's last does not flip the sides).
  Frozen maps use the same mix as live hunt: all-account crowding unless
  Binance crowd vs **top-trader size** are opposite, then size wins.
  The **live** map then shrinks 10x/25x/50x/100x bands by that later-print
  hit rate so weak levels are shorter and faded. On bands that did print,
  once there are at least three hits, the map also multiplies by `sizeScale`:
  realized notional divided by predicted notional on those hits, pulled in
  fully after twelve hits, never above 1, and never below 0.15. A quiet
  window is not counted again in that amount. The hit-rate confidence still drives
  `liquidation_level` alerts and tags `liquidation_notional` fires with
  `importance` (see [`price-alerts.md`](price-alerts.md)).
  A quiet window is a miss only when the liquidation feed covers the
  look-ahead; pending / missing tape is not a miss. Combined sums
  independent venues.
  Market-wide `symbol=all` is stored-print time bars for the requested range (12h through 30d). A zero bar was quiet or was not recorded. Prints still only in memory are added on top.
- Max pain (`GET /liquidation-max-pain`): largest estimated pocket **above**
  (shorts) and **below** (longs) last price. Not the hunt target.
  See [`liquidation-max-pain.md`](liquidation-max-pain.md). Web:
  `/liquidations?view=max-pain`.
- Hypothetical “what if spot is walked to force liquidations” is a separate model, including up vs down ease scores: [`liquidation-hunt.md`](liquidation-hunt.md). Web tab: `/liquidations?view=hunt`.
- Cascade detector: `GET /api/v1/market/liquidation-cascade`
  - Compares the last **1m / 5m / 15m** long and short notional to that stream's own typical (median of prior blocks over ~6 hours).
  - `grade` is `quiet` / `elevated` (≥2×) / `cascade` (≥4× + enough hits) / `extreme` (≥8×).
  - Binance, Bybit, and OKX are scored **separately**. `both.agree` is true only when the **same side** is `cascade` or hotter on all three. A quiet venue keeps it off.
  - `symbol=all` pools every tracked coin (market-wide risk). `GET /liquidation-cascade/scan` adds ranked bursting coins.
  - `episodes` lists each wave in the last 24h: when it started, how long it lasted, long vs short notional, and the price move. A still-running wave has `open=true`. Overlapping same-side waves on two or more venues become one `combined` episode. Combined rows name `startedBy` (first print), `venueStarts[].lagSec` (how many seconds later the others followed), and `venueStarts[].totalNotional` / `sharePct` (how much of the wave that venue produced). `largestBy` is the venue with the most notional. If two venues share the same first-print second, `orderUnknown` is true and the summary says they started together.

## Cascade grades

| Grade | Meaning |
|---|---|
| `quiet` | Current burst is in a normal range for that stream |
| `elevated` | About 2× typical, at least two hits |
| `cascade` | About 4× typical with enough prints in the window |
| `extreme` | About 8× typical with a larger hit count |

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation.go`, `liquidation_desk.go`, `liquidation_history.go`, `liquidation_levels.go`, `liquidation_levels_review.go`, `liquidation_cascade.go`, `liquidation_cascade_episode.go` |
| Adapters | `adapter/binance/liqhub.go`, `adapter/binance/liqhist.go`, `adapter/bybit/liqhub.go`, `adapter/bybit/liqhist.go`, `adapter/futuresstore/liquidation_desk.go` |
| Service | `GetLiquidations`, `GetLiquidationOverview`, `GetLiquidationLevels`, `GetLiquidationCascade`, `ScanLiquidationCascades`; stored desk totals in `service/market/liquidation_desk.go` and `service/futureshist/liquidation_desk.go`; reconnect fill in `service/futureshist/backfill.go` |
| HTTP | `GET /api/v1/market/liquidations`, `/liquidations/overview`, `/liquidation-levels`, `/liquidation-cascade`, `/liquidation-cascade/scan`, `/liquidation-max-pain` |
| MCP / AI | `get_liquidations`, `get_liquidation_overview`, `get_liquidation_levels`, `get_liquidation_cascade`, `scan_liquidation_cascades`, `get_liquidation_max_pain` |
| Web | `/liquidations` — cards + treemap; **Chart** tab; **Cascade** tab; **Max pain** tab; Hunt; Heatmap |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/futureshist/ ./internal/service/market/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT&exchange=binance"
curl "http://localhost:8080/api/v1/market/liquidations/overview?window=24h&limit=20"
curl "http://localhost:8080/api/v1/market/liquidation-levels?symbol=BTCUSDT&exchange=all&range=24h"
curl "http://localhost:8080/api/v1/market/liquidation-levels?symbol=all&exchange=binance"
curl "http://localhost:8080/api/v1/market/liquidation-cascade?symbol=BTCUSDT&exchange=all"
curl "http://localhost:8080/api/v1/market/liquidation-cascade?symbol=all"
curl "http://localhost:8080/api/v1/market/liquidation-cascade/scan"
curl "http://localhost:8080/api/v1/market/liquidation-max-pain?symbol=BTCUSDT"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- 24h is complete only after that coin has had live or backfilled coverage for 24h on the requested venue(s). A newly tracked coin, or a hole the venue cannot fill from history, stays `complete=false` even if the server is older.
- Binance public feed is a 1s largest-hit sample, not every fill.
- History fill uses Binance `GET /fapi/v1/allForceOrders` and Bybit `GET /v5/market/recent-liquidation` when those public endpoints answer. If they do not, the gap stays and `missingSeconds` shows the remaining hole.
