# Price correlation vs BTC and ETH

## Goal

Alts usually follow BTC (and often ETH), but sometimes they decouple or
follow a few minutes later. For any coin, show how similar its recent
movement is to BTC and ETH.

## Behavior

`GET /api/v1/market/correlation?symbol=SOLUSDT`

Windows:

| Window | Bars used |
|---|---|
| 1h | 1-minute closes |
| 4h | 5-minute closes |
| 24h | 5-minute closes |

For each window vs **BTC** and vs **ETH**:

- `corr` — Pearson correlation of bar-to-bar percent returns (−1 to +1)
- `beta` — typical coin move when the reference moves 1%
- `sameDirPct` — share of bars that went the same way
- `relation` — `follows` (≥0.70) / `loose` (≥0.40) / `mixed` / `inverse` (≤−0.40)
- `timing` — `together`, `lags` (follows later), or `leads`
- window `%` move for the coin, BTC, and ETH
- a short summary

Default venue is **Binance** spot (same quote as the pair; USDT fallback).
`exchange=bybit|coinbase` is allowed. Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/correlation.go` |
| Service | `backend/internal/service/market/correlation.go` |
| HTTP | `GET /api/v1/market/correlation` |
| MCP / AI | `get_price_correlation` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Correlation
curl "http://localhost:8080/api/v1/market/correlation?symbol=SOLUSDT"
```
