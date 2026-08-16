# Whale trades

## Goal

Show **very large buys, sells, longs, and shorts** so a huge print on a
small-cap coin is easy to notice. Sort **biggest first**. Each row has
average price, first and last trade time, and total size.

## Behavior

`GET /api/v1/market/whales?symbol=BTCUSDT`

- Clusters consecutive same-side futures fills within 2 seconds (VWAP)
- `side` `buy` / `sell` is the taker; futures buy = aggressive **long**,
  sell = aggressive **short**
- Adds large **liquidations** from the live book (last hour)
- Sorted by notional descending
- `unusual` is true when notional is at least **0.05% of circulating
  market cap** (small mcap + huge trade)
- Omit `symbol` to scan the top ~25 liquid USDT pairs

Optional: `exchange=binance|bybit|all` (default `all`),
`minNotional` (default 100000), `limit` (default 30).

Tape is the newest ~1000 prints per coin (minutes, not 24h).
Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/whales.go` |
| Service | `backend/internal/service/market/whales.go` |
| HTTP | `GET /api/v1/market/whales` |
| MCP / AI | `get_whale_trades` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/transport/http/... -run Whale
curl "http://localhost:8080/api/v1/market/whales?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/whales"
```
