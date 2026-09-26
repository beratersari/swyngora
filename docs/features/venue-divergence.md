# Venue divergence (Binance, Bybit, OKX)

## Goal

See which futures books are moving together and which one is moving apart.
Positioning already includes OKX USDT-swap, so the comparison uses the same
three venues: Binance USD-M, Bybit linear, and OKX.

## Behavior

`GET /api/v1/market/venue-divergence?symbol=BTCUSDT`

Compares four signals we already have. A venue that did not print is skipped.
Two venues are still enough to compare.

| Metric | Same | Different |
|---|---|---|
| Open interest change (primary window) | both up or both down | one up, one down |
| Funding payer | same side pays | longs pay vs shorts pay |
| Account crowding | both long or both short | long-crowded vs short-crowded |
| Price + OI regime | same lean | bullish vs bearish |

OKX does not publish account long/short, so it stays out of the crowding row.

`groups` lists the exchanges that share a bullish or bearish lean.
`outliers` names the smaller side when the other venues agree. A one-versus-one
split has groups and no single outlier.

`alignment`:

- `same` — the venues that have a direction share it
- `opposite` — one lean against the other
- `mixed` — they differ, but not a clean bullish-versus-bearish split
- `unknown` — fewer than two venues printed

Each `diffs[]` row has `readings` (one value per venue), `same`, `different`,
and `whyItMatters`.

Web: coin detail **Structure** tab.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/venue_divergence.go` |
| Service | `backend/internal/service/market/divergence.go` |
| HTTP | `GET /api/v1/market/venue-divergence` |
| MCP / AI | `get_venue_divergence` |
| Web | `frontend/src/components/organisms/VenueDivergenceBoard` on the Structure tab |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -count=1 -run 'CompareVenue|GetVenueDivergence'
curl "http://localhost:8080/api/v1/market/venue-divergence?symbol=BTCUSDT"
```

## Limits

- Crowding is only on venues that publish account long/short (Binance and Bybit).
- A venue with an error is left out of the comparison.
- Informational only.
