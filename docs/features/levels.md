# Support and resistance

## Goal

Find **support** and **resistance** areas for any coin using **price
history, volume, and the live order book** together. Show how far each
area is from last price, how many times it was tested, how much buy/sell
liquidity sits there, and — when price is close to or through a level —
how strong a breakout looks from volume, book, and taker flow.

## Behavior

`GET /api/v1/market/levels?symbol=BTCUSDT`

- Clusters recent 1h swing highs/lows and volume into price zones (~0.35% bins)
- `supports` / `resistances`: mid, band, `distancePct`, `tests`, quote `volume`
  of touches, `bidNotional` / `askNotional` on the live book near the zone
- `active`: the nearest zone being approached, tested, or broken
- `breakout`: `approaching` | `testing` | `broken` with a 0–100 score from
  recent volume vs typical, whether the book is thin or thick, and taker
  buy/sell (1h)

Default venue is Binance spot. Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/levels.go` |
| Service | `backend/internal/service/market/levels.go` |
| HTTP | `GET /api/v1/market/levels` |
| MCP / AI | `get_support_resistance` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Levels
curl "http://localhost:8080/api/v1/market/levels?symbol=BTCUSDT"
```
