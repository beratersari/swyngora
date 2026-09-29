# Durable futures history

## Goal

Keep **open interest**, **funding rate**, **long/short ratio**, and **liquidation**
history for Binance USD-M, Bybit linear, and OKX USDT-swap so values survive a
server restart and can be queried later by users or the AI.

## Behavior

- Background worker (default every **5 minutes**) samples configured + recently
  requested symbols on **each venue independently**. If one venue errors, the
  others are still saved.
- Liquidation events are written as they arrive on the websocket (async queue,
  flushed on shutdown; overflow writes synchronously). The last **24 hours**
  are reloaded into the in-memory book on startup, **per venue**, with
  coverage clocks so rolling windows stay complete after a restart.
- Duplicates are ignored:
  - snapshots: `(metric, exchange, symbol, sampled_at, predicted)`
  - liquidations: `(exchange, symbol, side, time, price, quantity)`
- Snapshot times are floored to 5 minutes for OI / long-short / predicted funding.
  Settled funding uses the venue settlement timestamp.
- Long/short rows store the all-account print and, on Binance, **top-trader
  position** (`sizeLongPct`) and top-trader account (`topAccLongPct`) shares
  so later hunt / liquidation-level replays can use size when the crowd
  leans the other way.
- Rows older than `FUTURES_HISTORY_RETENTION` (default 90 days) are purged.
- Startup pages public 1-hour open interest, funding, and long/short back
  to that window (at most 90 days) for the seed pairs. A later start skips
  a series whose earliest row already reaches the back of the window.
- Liquidation prints are kept for the same retention. Public REST only
  refills about the last 7 days (Binance rejects a wider force-order
  window). Older prints are the ones the websocket already stored.

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
| `FUTURES_HISTORY_RETENTION` | `2160h` (90 days) |
| `FUTURES_HISTORY_SYMBOLS` | extra CSV pairs (BTC/ETH/SOL/… always included) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/futuresstore/ ./internal/service/futureshist/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/futures-history?metric=open_interest&symbol=BTCUSDT"
```
