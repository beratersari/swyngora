# Market order count ratio

## Goal

Print size is how large each fill is. Repeated size is one clip coming
back. This read is **how many** aggressive prints were buys versus sells.

## Behavior

`GET /api/v1/market/order-count?symbol=BTCUSDT`

- Windows **1m / 5m / 15m**
- `buyCount` and `sellCount` are aggressive futures prints
- `ratio` is (buy − sell) / (buy + sell), from **−1** (all sells) to **+1** (all buys)
- `lean` is buy, sell, or balanced when the ratio is inside ±0.10
- `exchange=all` sums the counts, then divides. It does not average the ratios

| Venue | Source |
|---|---|
| Binance USD-M | Paged `aggTrades` |
| Bybit linear | Newest ~1000 public trades |
| OKX USDT-swap | Newest public trades |

Informational only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/order_count.go` |
| Service | `GetOrderCount` |
| HTTP | `GET /api/v1/market/order-count` |
| MCP / AI | `get_order_count_ratio` |
| Web | Coin detail futures tape, above repeated size |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -count=1 -run OrderCount
curl "http://localhost:8080/api/v1/market/order-count?symbol=BTCUSDT"
```
