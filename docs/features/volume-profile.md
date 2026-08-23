# Volume profile (volume by price)

## Goal

See **where** trading happened by **price**, not only how much traded over
time. BTC can travel 64k–68k while most volume sits around 65200 and some
prices are almost empty.

Useful when:

- you want the **point of control** (the price with the most volume)
- you want the **value area** (the main band that collected most volume)
- you want **buy vs sell** at those prices, per venue and combined

## Behavior

`GET /api/v1/market/volume-profile?symbol=BTCUSDT`

- Works for BTC and any other spot pair the venues list
- `exchange=all` (default): **Binance** and **Bybit** separately plus `combined`
- Time range: `window=1h|4h|24h|7d|30d` (default `24h`) **or** `startTime` +
  `endTime` (RFC3339 or unix ms). Max 30 days
- Optional `tickSize` (price row width, e.g. `50`). Omit to pick a row size
  that keeps the profile readable (widened if it would exceed 200 rows)
- Each bin: `price`–`high`, total quote volume, `buyVolume` / `sellVolume`,
  share of total, `isPoc`, `inValueArea`, `isHvn`
- **POC:** the row with the most volume (ties go to the row closer to last)
- **Value area:** expand from the POC to neighboring rows until about **70%**
  of volume is inside. `valueArea.low` / `high` is that band
- `lastVsValueArea`: `above` | `inside` | `below`
- Combined **adds** both venues at the same prices and shows `shares` on
  each row. Combined buy/sell is **partial** when Bybit has no taker-buy

Volume is **quote notional** (USDT). Built from **spot candles**: each bar’s
volume is spread evenly across the prices it traded (`high`–`low`). That is
an approximation, not tick-level prints. Binance klines include taker-buy, so
buy/sell is available there. Bybit klines do not.

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/volume_profile.go` |
| Service | `backend/internal/service/market/volumeprofile.go` |
| HTTP | `GET /api/v1/market/volume-profile` |
| MCP / AI | `get_volume_profile` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run VolumeProfile
curl "http://localhost:8080/api/v1/market/volume-profile?symbol=BTCUSDT&window=24h"
curl "http://localhost:8080/api/v1/market/volume-profile?symbol=ETHUSDT&exchange=binance&window=4h&tickSize=5"
```
