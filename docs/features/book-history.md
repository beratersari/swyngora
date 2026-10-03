# Order-book history

## Goal

Keep regular snapshots of the **live spot order book** so you can see what
the book looked like **before a strong price move** and how it **changed
after**. Compare two times to see which price levels gained or lost
liquidity.

## Behavior

A background worker samples majors plus recently viewed pairs on Binance,
Coinbase, and Bybit independently. The full ladder is stored every **1 minute**.
Between those minutes the same books are read about every **5 seconds**. A wall
that shows up and leaves before the next ladder (for example it rests 10 seconds)
is stored as a **brief** mark and returned on that minute's snapshot. Other
resting size that is not a wall is kept too, when the change is at least 3% of
that side and the minute ladder does not show it: size that leaves, or a swing
whose peak (or trough) is gone by the snapshot. Each sample
stores:

- grouped **bid/ask levels** (top 25)
- **spread** and mid
- **total bid/ask liquidity** inside ±2%
- **imbalance** / pressure
- large **walls**
- **brief walls** that did not survive until the minute sample (`flashes`)
- **size moves** that are not walls and did not survive on the ladder (`moves`)

Duplicates for the same venue + pair + minute are ignored. Rows older than
`ORDERBOOK_HISTORY_RETENTION` (default 30 days) are purged.

`GET /api/v1/market/orderbook/history?symbol=BTCUSDT&at=2026-08-16T12:00:00Z`

Returns the stored book **nearest** that time (prefers at-or-before).

`GET /api/v1/market/orderbook/history?symbol=BTCUSDT`

Lists recent samples (newest first; summaries only).

`GET /api/v1/market/orderbook/history/compare?symbol=BTCUSDT&from=...&to=...`

Loads the two nearest books and lists levels that **gained** or **lost**
liquidity, plus mid/spread/imbalance change and walls that appeared or were
pulled.

Times are RFC3339 or unix milliseconds. Informational only — not financial
advice.

The coin order-book heatmap uses these ladders, flashes, and moves when the
window is longer than 30 minutes (`1h`, `4h`, `24h`, `3d`, `7d`). Each column
is a minute, or a group of minutes on a multi-day window. A flash or size
move raises the notional at that price for the bucket it happened in.

To split a size drop into **traded vs pulled vs added** (8M bid at 65500
down to 2M), use [`book-liquidity-flow.md`](book-liquidity-flow.md).

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/book_history.go` |
| Store | `backend/internal/adapter/bookhiststore/` |
| Service | `backend/internal/service/bookhist/` + `service/market/bookhist.go` |
| HTTP | `GET /api/v1/market/orderbook/history`, `.../history/compare` |
| MCP / AI | `get_orderbook_history`, `compare_orderbook_history` |

## Config

| Variable | Default |
|---|---|
| `ORDERBOOK_HISTORY_DB_PATH` | `data/orderbook.db` |
| `ORDERBOOK_HISTORY_INTERVAL` | `1m` |
| `ORDERBOOK_HISTORY_RETENTION` | `168h` |
| `ORDERBOOK_HISTORY_SYMBOLS` | extra CSV pairs (BTC/ETH/SOL/… always included) |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/bookhiststore/ ./internal/service/bookhist/ ./internal/service/market/ -run BookHist
curl "http://localhost:8080/api/v1/market/orderbook/history?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/orderbook/history?symbol=BTCUSDT&at=2026-08-16T12:00:00Z"
```
