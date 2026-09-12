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

`why` is a plain-language story of **why the move may have happened**
(before / during / after, plus similar past cases). It interprets the
tape — volume vs typical, who was aggressive, book thinning, open
interest vs price, funding, liquidations, sweeps, and unusual listed
options (OI / put-call / implied vol) on BTC and ETH — instead of
listing raw metrics.

`timeline` is the same tape on **one clock**. Each beat is a start
print (open interest, takers, price on Binance or Bybit, liquidations,
listed options, a sweep or absorption) with the time, how far it sits
from `at` (`offset` / `offsetLabel`), and which venue printed it.
Missing series are omitted. Informational only.

On BTC and ETH, each phase can include stored Deribit desk samples
(`options`): listed open interest, put/call lean, DVOL/ATM IV, and
`unusual` when that series jumped in **this** window. `started` is
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
a start time. The first signal across price, open interest, and taker
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
| Web | `/forensics` — `frontend/src/components/pages/ForensicsPage/` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run 'Forensics|Around'
curl "http://localhost:8080/api/v1/market/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z"
curl "http://localhost:8080/api/v1/market/forensics/scan?limit=12&lookback=3d"
curl "http://localhost:8080/api/v1/market/forensics/radar?since=7d"
```

Open `/forensics`, pick a symbol and time, confirm the why-story, the
time-ordered timeline, three phase cards, and the similar-move table
(or a soft similar error).
