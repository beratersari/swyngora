# Volume surge (vs typical)

## Goal

Raw 24h volume is not enough. A coin that usually does **2M in 5
minutes** and suddenly does **10M** is a different story. Show **how
many times typical** the latest 5m / 15m / 1h is, **buy vs sell**
separately, and **which coins** are running that hot.

## Behavior

`GET /api/v1/market/volume-surge?symbol=BTCUSDT`

- Windows **5m / 15m / 1h**
- `current` vs `typical` (median of the prior ~24 hours of that window)
- `ratio` = current / typical
- `buyRatio` / `sellRatio` when the venue publishes taker-buy (Binance
  klines). Bybit is total-only
- `dominant`: which side the extra volume is coming from
- `grade`: `typical` | `elevated` (≥1.5x) | `high` (≥3x) | `extreme` (≥6x)

`GET /api/v1/market/volume-surge/scan`

- Scans top 24h-volume pairs (`limit` default 30, max 50)
- Keeps coins with `maxRatio >= minRatio` (default **2**)
- Ranked hottest first
- Default `exchange=binance` so buy/sell is available

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/volume_surge.go` |
| Service | `backend/internal/service/market/volumesurge.go` |
| HTTP | `GET /api/v1/market/volume-surge`, `.../volume-surge/scan` |
| MCP / AI | `get_volume_surge`, `scan_volume_surges` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run Surge
curl "http://localhost:8080/api/v1/market/volume-surge?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/volume-surge/scan?minRatio=2"
```
