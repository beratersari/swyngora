# VWAP (volume-weighted average price)

## Goal

Pick a **start time** and see the average price from then until now,
but **not** a simple average. Prices with **more trading volume** pull
the result more. Little volume at 100 and a lot at 110 should land
**closer to 110**. Also show how far **last** is above or below that
average, and **total volume** since the start.

## Behavior

`GET /api/v1/market/vwap?symbol=BTCUSDT`

- `exchange=all` (default): **Binance** and **Bybit** separately plus
  `combined` (each venue weighted by its quote volume)
- Time range: `window=1h|4h|24h|7d|30d` (default `24h`) **or**
  `startTime` (+ optional `endTime`, default now). Max 30 days
- Each candle uses typical price `(high+low+close)/3` × quote volume
- `vwap`, `lastPrice`, `distance`, `distancePct`, `side`
  (`above` / `below` / `at`), `volume`, `barCount`
- Combined `shares` show how much volume each venue contributed

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/vwap.go` |
| Service | `backend/internal/service/market/vwap.go` |
| HTTP | `GET /api/v1/market/vwap` |
| MCP / AI | `get_vwap` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run VWAP
curl "http://localhost:8080/api/v1/market/vwap?symbol=BTCUSDT&window=24h"
curl "http://localhost:8080/api/v1/market/vwap?symbol=ETHUSDT&startTime=2026-08-20T00:00:00Z"
```
