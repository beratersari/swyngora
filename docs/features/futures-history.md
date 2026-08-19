# Durable futures history

## Goal

Keep **open interest**, **funding rate**, **long/short ratio**, and **liquidation**
history for Binance USD-M and Bybit linear so values survive a server restart
and can be queried later by users or the AI.

## Behavior

- Background worker (default every **5 minutes**) samples configured + recently
  requested symbols on **each venue independently**. If Bybit errors, Binance
  is still saved (and the reverse).
- Liquidation events are written as they arrive on the websocket (async queue)
  and reloaded into the in-memory book on startup (last 24h).
- Duplicates are ignored:
  - snapshots: `(metric, exchange, symbol, sampled_at, predicted)`
  - liquidations: `(exchange, symbol, side, time, price, quantity)`
- Snapshot times are floored to 5 minutes for OI / long-short / predicted funding.
  Settled funding uses the venue settlement timestamp.
- Rows older than `FUTURES_HISTORY_RETENTION` (default 30 days) are purged.

`GET /api/v1/market/futures-history?metric=open_interest&symbol=BTCUSDT`

`metric`: `open_interest` | `funding` | `long_short` | `liquidations`

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/futures_history.go` |
| Store | `backend/internal/adapter/futuresstore/` |
| Service | `backend/internal/service/futureshist/` |
| HTTP | `GET /api/v1/market/futures-history` |
| MCP / AI | `get_futures_history` |

## Config

| Variable | Default |
|---|---|
| `FUTURES_HISTORY_DB_PATH` | `data/futures.db` |
| `FUTURES_HISTORY_INTERVAL` | `5m` |
| `FUTURES_HISTORY_RETENTION` | `720h` |
| `FUTURES_HISTORY_SYMBOLS` | extra CSV pairs (BTC/ETH/SOL/… always included) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/futuresstore/ ./internal/service/futureshist/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/futures-history?metric=open_interest&symbol=BTCUSDT"
```
