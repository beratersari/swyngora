# Feature: Futures volume share by exchange

## Problem / goal

Show how BTC (or another coin) **futures volume** splits across venues —
e.g. **55% Binance, 30% Bybit, 15% OKX** — and whether those shares
**changed in the last hour** (who gained, who lost).

## Behavior

`GET /api/v1/market/volume-share?symbol=BTCUSDT`

- `exchange=all` (default): Binance USD-M, Bybit linear, OKX USDT-swap
- Windows: **15m**, **1h**, **4h**
- Each window:
  - `venues[].sharePct` — percent of that window's quote volume
  - `venues[].volume` — USDT quote volume
  - `priorSharePct` / `changePct` — mix vs the **same length just before**
    (1h now vs the hour before)
  - `kind` — `gained` / `lost` / `stable` (≥ 1 percentage point)
  - `gained` / `lost` — venue names that moved

Source: 5-minute perpetual **quote-volume** klines (not spot). Informational
only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/volume_share.go` |
| Adapters | `adapter/binance/futures_volume.go`, `bybit/futures_volume.go`, `okx/futures_volume.go` |
| Service | `GetVolumeShare` |
| HTTP | `GET /api/v1/market/volume-share` |
| MCP / AI | `get_volume_share` |
| Web | Coin detail futures tape (`TapePanel`) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/adapter/okx/ ./internal/service/market/ ./internal/transport/http/handler/ -count=1 -run VolumeShare
curl -s "http://localhost:8080/api/v1/market/volume-share?symbol=BTCUSDT" | jq ".windows[] | select(.window==\"1h\")"
```

## Limits

- Quote volume from 5m klines, not taker buy/sell (`get_taker_flow`).
- OKX uses `volCcyQuote` on USDT-swap candles.
- A missing venue is omitted from the mix (shares renormalize to the venues that returned bars).
- Informational only — not financial advice.
