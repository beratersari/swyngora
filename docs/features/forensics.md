# Market Forensics

## Goal

One desk for a **coin + time**: what happened **before**, **during**, and
**after** that move, using the existing around tape (price, volume,
takers, VWAP, vs typical, POC, sweeps, stored book / open interest /
funding / liquidations, and **listed BTC/ETH options** when those
samples exist). When possible, also show **similar past
moves** and how price moved after those.

## Behavior

`GET /api/v1/market/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z`

Web: `/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z`

On a crypto coin chart (`/markets/binance/BTCUSDT`), clicking a candle
opens this same read under the chart. `at` is that candle's open.
The window stays the forensics default: one hour before the open, then
15 minutes. The card shows the why-story, the timeline, and what
similar setups did next. A row in that table moves the read to that
past time. Clear, or the same candle, removes it. The link opens
`/forensics` with the same symbol, time, `exchange=all`, `window=1h`,
and `during=15m`. Nasdaq and BIST charts do not call it. The click
still restarts CVD and the positioning lines when those layers are on.

This is a composition of existing analysis — it does not rebuild the
tape:

| Piece | Source |
|---|---|
| Before / during / after | `GET /api/v1/market/around` |
| Similar past setups + after horizons | `GET /api/v1/market/around/similar` with the same `at` |

`at` is required (RFC3339 or unix ms). Same `window` / `during` /
`exchange` as `/around`. Same similar knobs as `/around/similar`
(`lookback`, `fields`, `weights`, `minCoverage`, `horizons`).

If similar-move search fails, `tape` is still returned and
`similarError` explains why.

Each similar match lists **which tape pieces matched** (`whySimilar`
plus per-field `compared` scores for price, volume, taker flow, open
interest, the order book, and listed options when both sides have
samples). `used` / `missing` / `matches` name those fields. Coverage
is how much of the selected weight actually had data; similarity is
scored only on fields that were present.

`why` is a plain-language story of **why the move may have happened**
(before / during / after, plus similar past cases). It interprets the
tape — volume vs typical, who was aggressive, book thinning, open
interest vs price, funding, liquidations, sweeps, unusual listed
options (OI / put-call / implied vol), dealer gamma / flip vs
spot on BTC and ETH, and the listed futures curve (contango /
backwardation and the near-vs-far gap) — instead of listing raw metrics.

`tapeRead` on each venue (and combined) is a flow-at-prices **summary** of
the before + during window: where **most volume** printed (POC) with
market buy vs sell there, where **buyer/seller dominance** started,
and if **delta flipped** while price was still going the other way
(for example delta turned negative after 65800 while price was still
rising). The why-story and a timeline `tape` beat use that same read
so the AI tools see the numbers too. The **per-price ladder** (buy,
sell, print count, delta at 65520 while price traded 65400–65600) is
`GET /api/v1/market/flow-at-prices` — see [`flow-at-prices.md`](flow-at-prices.md).

`timeline` is the same tape on **one clock**. Each beat is a start
print (open interest, takers, price on Binance or Bybit, liquidations,
listed options, dealer gamma / flip vs spot, the futures curve, a sweep or absorption) with the time, how far it sits
from `at` (`offset` / `offsetLabel`), and which venue printed it.
Missing series are omitted. The clock only includes prints that were
**already known** at the observer time (`timeline.knownUntil`).

Optional `asOf` (RFC3339 or unix ms) is that observer clock — tape and
timeline stop there. Example: a 14:00 move checked `asOf=14:10` does
not show a 14:30 liquidation or a Bybit start at 14:30. Checking again
later (or passing a later `asOf`) can add those prints.

When `asOf` is omitted it is **now**. An old event reviewed later keeps
prints that have already happened (a 14:30 liquidation stays on the
clock). A 14:00 move checked at 14:10 still hides 14:30.

The coin-chart click and the event view on `/forensics` send `asOf` at the end of the during window once that end is in the past. The minutes of the move stay in the card. Prints after that end stay out. A window that has not finished yet omits `asOf`, so the server clock stays now.

A venue whose candles (or OI / taker bars) are **missing until later**
does not get a start at that first sample. If Binance moved at 14:05
and Bybit data only appears at 14:08, the tape does not say Bybit
followed at 14:08 — that print is missing data, not a follow. A later
impulse after those late bars are already on the tape can still be a
follow.

A **hole in the middle** of the tape is the same: if bars stop and the
next print is already the move, we do not date the start (it may have
begun in the gap).

Timestamps are **bar or sample windows**, not exact ticks. A 5m candle
at 14:00 means sometime in that bar. An OI (or options) sample at
14:05 is as-of that clock — the change may have happened anywhere
since the previous print (14:00–14:05), so a 1m price bar at 14:03 is
not “price first.” If two stamps overlap at that size — 5m vs 5m two
minutes apart, 5m vs 1m inside the same bar, or a snapshot window vs a
finer candle — `venueTiming` says they started around the same time
and the timeline marks `orderUnknown`. The summary does not narrate a
sequence we cannot know. Liquidation prints stay exact.

Informational only.

On BTC and ETH, each phase can include stored Deribit desk samples
(`options`): listed open interest, put/call lean, DVOL/ATM IV, dealer
gamma (long / short / flip vs spot) when those samples exist, and
`unusual` when that series jumped in **this** window. The timeline
adds a **gamma** beat for the before-window state — whether dealers
were long or short gamma going into the move, and whether spot sat
above or below the flip. It also adds a **curve** beat — whether
listed dated futures were in contango or backwardation going into
the move, and how much richer the far (or near) contract was. `started` is
`before` when the print was already underway at the price event, or
`during` when it only appeared once price was moving. If it started
before, `lead` / `leadLabel` say how much earlier. A 4h or 24h unusual
flag is only the lookback — the start is the first sample that left
the quiet baseline, which may be later in that window.
`after` says whether options kept going the same way once price
started, turned the other way, or went quiet. When both venues
are present, `venueTiming` says whether they started together, which
one printed first, and whether the other followed later or the move
stayed on one venue. A venue that did not actually move is not given
a start time. A venue whose tape is missing until later is not given
a start at that first sample. The first signal across price, open interest, and taker
flow is the `leader` (`leaderMetric` is `price` / `oi` / `takers`) —
price-only can be misleading if the other venue's OI or takers moved
earlier. An earlier OI or taker print is only treated as a precursor
when it **kept going until price**; a spike that faded or reversed is
labeled that way and is not the leader. If the last OI sample or
taker bar is much earlier than the price start (or is only the start
print), the path in between is unknown — `until` is omitted, and the
tape does not say continued, faded, or reversed. The taker bar at the
price start is the first minute of the move, not pre-price tape.
Continued / faded / reversed also need a continuous pre-price taker
path at the tape's bar size (1m, 5m, …): no hole longer than one
missing bar, the last bar before price must exist, and there must be
tape after the start print — one bar only shows the flow started.
A last print near 12:00 does not fill a multi-bar gap after the start. If one venue printed first,
`beforeFollow` uses that venue's real start (for example 12:05), not
the during-window close (12:15).
`together` is how
many of OI / put-call / IV moved at once; two or three series
together is treated as more important than a single print
(`weight` `pair` / `cluster`). A missing start or end sample
is omitted — it is not treated as a jump from zero. Opposite options
moves are not scored as similar. Other coins omit the block.

`/around*` endpoints stay as they are. Optional `at` on
`/around/similar` compares the tape **before that time** instead of
the current last window.

## Current market vs past setups

`GET /api/v1/market/forensics/scan`

Web: `/forensics?view=now`

Scans top quote-volume coins and compares each coin's **current**
tape to the setup **before that coin's own** important past moves.
Hits are ranked by closeness. Each row says **why** the current tape
looks like the closest past setup (`whySimilar` plus per-field
`compared` scores for price, volume, taker flow, open interest, the
order book, and listed options when both sides have samples) and what usually happened after those similar cases
(`afterStory`, `lean`, `afterHorizons`).
Default 12 coins, 3d lookback. Click a coin to open the event tape.
**Create alert** on a row (or “any coin”) opens `/alerts?kind=setup_radar` so you get notified when that tape looks similar enough again (`kind=setup_radar`, MCP `create_setup_radar_alert` — see [`price-alerts.md`](price-alerts.md)).
MCP: `scan_market_forensics`.

## Saved hits and usefulness

`GET /api/v1/market/forensics/radar`

Web: `/forensics?view=review`

When a setup appears (Now scan or the background worker), it is saved
once per coin per 4 hours. After 15m / 1h / 4h the later price is
filled in. Review shows how often an up/down lean matched that later
move (`useful`), plus median after-return. Click a row to open the
event tape at found time. MCP: `review_setup_radar`.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/around_forensics.go`, `around_forensics_timeline.go`, `around_options.go`, `radar_case.go` |
| Service | `backend/internal/service/market/around_forensics.go` |
| HTTP | `GET /api/v1/market/forensics`, `GET /api/v1/market/forensics/scan`, `GET /api/v1/market/forensics/radar` |
| MCP / AI | `get_market_forensics`, `scan_market_forensics`, `review_setup_radar` |
| Web | `/forensics` — `frontend/src/components/pages/ForensicsPage/`; coin chart click — `frontend/src/components/organisms/CoinCandleForensics/` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run 'Forensics|Around'
curl "http://localhost:8080/api/v1/market/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z"
curl "http://localhost:8080/api/v1/market/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z&asOf=2026-08-20T14:10:00Z"
curl "http://localhost:8080/api/v1/market/forensics/scan?limit=12&lookback=3d"
curl "http://localhost:8080/api/v1/market/forensics/radar?since=7d"
```

Open `/forensics`, pick a symbol and time, confirm the why-story, the
time-ordered timeline, three phase cards, and the similar-move table
(or a soft similar error). Click a similar row to load that past
event's timeline and before / during / after tape (same symbol,
window, and during).

On `/markets/binance/BTCUSDT`, click a candle. The card under the chart
states what changed around that open and what similar setups did next.
Clear, or the same candle, removes it. Open the full forensics desk
keeps the same symbol and time.
