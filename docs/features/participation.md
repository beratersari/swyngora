# Feature: Participation ratio (OI delta / volume)

## Problem / goal

Hyblock-style **participation**: how much of futures **volume** stuck as
new (or closed) **open interest**. 2B of volume with OI up 300M is **15%**
new positions; the rest is transfer between existing books.

## Behavior

`GET /api/v1/market/participation?symbol=BTCUSDT`

- `exchange=all` (default): Binance USD-M and Bybit linear separately plus `combined`
- Windows: **15m**, **1h**, **4h**
- Per window:
  - `volume` — USDT taker volume (buy + sell)
  - `oiDelta` — signed USDT open-interest change
  - `ratio` — `oiDelta / volume` (0.15)
  - `ratioPct` — the same as percent (15)
  - `transfer` — `volume − |oiDelta|` (floored at 0)
  - `lean` — `entering` (≥ +5%) / `exiting` (≤ −5%) / `transfer` (near 0)

Combined sums volume and OI delta, then recomputes the ratio, only when
every venue that was asked has a complete window. A venue without that
window blanks the combined ratio and is named in `missing`. It is not
counted as zero. OKX is not included (no taker-volume series on this path).

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/participation.go` |
| Service | `GetParticipation` |
| HTTP | `GET /api/v1/market/participation` |
| MCP / AI | `get_participation` |
| Web | Coin detail futures tape (`TapePanel`) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run Participation
curl -s "http://localhost:8080/api/v1/market/participation?symbol=BTCUSDT" | jq .
```

## Limits

- Volume is aggressive taker notional (same family as `get_taker_flow`), not candle quote volume.
- OI delta is USDT **value** change (same as the open-interest tape). Mark-to-market can move notional without new contracts.
- Bybit 4h volume is complete only after this process has been collecting tape.
- `|ratio|` can exceed 1 when OI value moves more than captured volume.
- Informational only — not financial advice.
