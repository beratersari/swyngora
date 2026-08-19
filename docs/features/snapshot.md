# Combined market snapshot

## Goal

Sometimes volume and open interest start to rise before price moves, taker
buys get stronger, or funding flips. Put **price, volume, market cap, open
interest, funding, long/short, and taker buy/sell** on one tape for any coin,
with current values and 1h / 4h / 24h changes.

## Behavior

`GET /api/v1/market/snapshot?symbol=SOLUSDT`

- `spot`: last price, 24h quote volume, circulating market cap, plus per-window
  price / volume / mcap change
- `venues`: Binance and Bybit futures — OI, funding, account long %, taker
  buy/sell for 1h and 4h (24h taker is not collected yet)
- `summary`: short read when volume or OI is building ahead of price

Volume change is this window versus the previous window of the same length.
Market-cap change follows price when circulating supply is unchanged.
Informational only.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/snapshot.go` |
| Service | `backend/internal/service/market/snapshot.go` |
| HTTP | `GET /api/v1/market/snapshot` |
| MCP / AI | `get_market_snapshot` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Snapshot
curl "http://localhost:8080/api/v1/market/snapshot?symbol=SOLUSDT"
```
