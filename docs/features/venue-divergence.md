# Venue divergence (Binance vs Bybit)

## Goal

Notice when **Binance USD-M** and **Bybit linear** disagree on the same coin
— for example OI rising and longs crowding on Binance while Bybit funds the
other way — without checking each exchange by hand.

## Behavior

`GET /api/v1/market/venue-divergence?symbol=BTCUSDT`

Compares four signals we already have:

| Metric | Same | Opposite |
|---|---|---|
| Open interest change (primary window) | both up or both down | one up, one down |
| Funding payer | same side pays | longs pay vs shorts pay |
| Account crowding | both long or both short | long-crowded vs short-crowded |
| Price + OI regime | same lean | bullish (buildup/covering) vs bearish (short buildup / long unwind) |

`alignment`:

- `same` — no important opposite
- `opposite` — leans disagree, or an important driver is opposite (`important=true`)
- `mixed` — some differ but not a clean opposite
- `unknown` — one venue missing

Each `diffs[]` row has `binance`, `bybit`, `alignment`, and `whyItMatters`.
`title` + `summary` are the short read.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/venue_divergence.go` |
| Service | `backend/internal/service/market/divergence.go` |
| HTTP | `GET /api/v1/market/venue-divergence` |
| MCP / AI | `get_venue_divergence` |

## How to verify

```bash
cd backend && go test ./internal/domain/ -run CompareVenue
curl "http://localhost:8080/api/v1/market/venue-divergence?symbol=BTCUSDT"
```
