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
- Long/short rows store the all-account print and, on Binance and OKX,
  **top-trader position** (`sizeLongPct`) and top-trader account
  (`topAccLongPct`) shares so later hunt / liquidation-level replays can use
  size when the crowd leans the other way. Bybit publishes the account ratio
  only.
- Rows older than `FUTURES_HISTORY_RETENTION` (default 90 days) are purged.
  That retention is how long **stored** rows are kept, including samples the
  process recorded itself. It is not a promise that every public history
  endpoint can backfill 90 days.
- Startup pages public 1-hour history for the liquid recorder list, seeds
  first and capped at 100, only as far as that endpoint documents, and never
  past 90 days. The first liquid-name refresh runs before the archive starts.
  A later start skips a series whose earliest row already reaches the back of
  **that** window. A Binance rate-limit pause stops the current pass and
  retries it, up to 24 passes, so a finished series is not requested again.
- The live sampler keeps that same list (`domain.LiquidRecorderLimit`). Spot
  quote volume picks the extra names. `GET /api/v1/market/history-span` reports
  the stored span per venue. See `docs/features/chart-desk.md`.

| Series | Endpoint | Public window this archive requests |
|---|---|---|
| Binance open interest | `GET /futures/data/openInterestHist` | Latest 30 days. The legacy page says the latest 1 month; the official USD-M connector says the latest 30 days. |
| Binance long/short | `GET /futures/data/globalLongShortAccountRatio`, `topLongShortAccountRatio`, `topLongShortPositionRatio` | Latest 30 days on each page. |
| Binance funding | `GET /fapi/v1/fundingRate` | No day cap in the docs. This archive requests 90 days. |
| Bybit open interest | `GET /v5/market/open-interest` | Back to the symbol's launch time. This archive requests 90 days. |
| Bybit long/short | `GET /v5/market/account-ratio` | Earliest start is 20 July 2020. This archive requests 90 days. |
| Bybit funding | `GET /v5/market/funding/history` | No day cap in the docs. This archive requests 90 days. |
| OKX open interest | `GET /api/v5/rubik/stat/contracts/open-interest-history` with `period=1H` | Latest 1,440 entries, which is 1,440 hours at 1H. |
| OKX funding | `GET /api/v5/public/funding-rate-history` | Up to three months, and not past 90 days. |
| OKX long/short | `GET /api/v5/rubik/stat/contracts/long-short-account-ratio-contract`, plus the top-trader account and position ratios, `period=1H` | Latest 1,440 entries on each endpoint. Top traders are OKX's top 5% by position value. |

- Liquidation prints are kept for the same retention. Public REST only
  refills about the last 7 days (Binance rejects a wider force-order
  window; Bybit recent-liquidation and OKX liquidation-orders are recent
  prints). Older prints are the ones the websocket already stored.

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

`GET /api/v1/market/unusual` reads this store to say whether funding, open-interest change, liquidation notional, and daily CVD are high or low versus the samples already kept. It does not change how far each endpoint is backfilled. See `docs/features/unusual-context.md`.

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
