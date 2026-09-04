# Market Forensics

## Goal

One desk for a **coin + time**: what happened **before**, **during**, and
**after** that move, using the existing around tape (price, volume,
takers, VWAP, vs typical, POC, sweeps, stored book / open interest /
funding / liquidations). When possible, also show **similar past
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
interest vs price, funding, liquidations, sweeps — instead of listing
raw metrics. Informational only.

`/around*` endpoints stay as they are. Optional `at` on
`/around/similar` compares the tape **before that time** instead of
the current last window.

## Current market vs past setups

`GET /api/v1/market/forensics/scan`

Web: `/forensics?view=now`

Scans top quote-volume coins and compares each coin's **current**
tape to the setup **before that coin's own** important past moves.
Hits are ranked by closeness. Each row says what usually happened
after those similar cases (`afterStory`, `lean`, `afterHorizons`).
Default 12 coins, 3d lookback. Click a coin to open the event tape.
MCP: `scan_market_forensics`.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/around_forensics.go` |
| Service | `backend/internal/service/market/around_forensics.go` |
| HTTP | `GET /api/v1/market/forensics`, `GET /api/v1/market/forensics/scan` |
| MCP / AI | `get_market_forensics`, `scan_market_forensics` |
| Web | `/forensics` — `frontend/src/components/pages/ForensicsPage/` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -run 'Forensics|Around'
curl "http://localhost:8080/api/v1/market/forensics?symbol=BTCUSDT&at=2026-08-20T14:00:00Z"
curl "http://localhost:8080/api/v1/market/forensics/scan?limit=12&lookback=3d"
```

Open `/forensics`, pick a symbol and time, confirm three phase cards
and the similar-move table (or a soft similar error).
