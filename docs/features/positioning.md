# Positioning (price + open interest)

## Goal

For any coin, say whether Binance and Bybit futures look like **long buildup**,
**short buildup**, **long unwinding**, or **short covering**, with a short
reason and a **combined market direction**.

This is the classic futures matrix: read price change and open-interest change
together. Funding and account long/short only **corroborate** the label.

## Matrix

| Price | Open interest | Regime | Meaning |
|---|---|---|---|
| Up | Up | `long_buildup` | New longs opening |
| Down | Up | `short_buildup` | New shorts opening |
| Down | Down | `long_unwinding` | Longs closing |
| Up | Down | `short_covering` | Shorts closing |
| Flat / mixed | — | `neutral` | No clear label |

## Behavior

`GET /api/v1/market/positioning?symbol=BTCUSDT`

- `exchange=all` (default): Binance + Bybit separately, plus `combined`.
- Windows: **1h**, **4h** (primary when available), **24h**.
- Price from 1h candles (ticker 24h as fallback). OI from venue history.
- `reasons` explain the matrix and any funding / long-short support.
- `combined`: OI-weighted general market regime, `agreement` = agree|mixed|single.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/positioning.go` |
| Service | `backend/internal/service/market/positioning.go` |
| HTTP | `GET /api/v1/market/positioning` |
| MCP / AI | `get_positioning` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Positioning
curl "http://localhost:8080/api/v1/market/positioning?symbol=BTCUSDT"
```

Read `venues[].regime`, `venues[].summary`, and `combined`.
