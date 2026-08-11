# Futures liquidations

## Goal

Show Coinglass-style **long vs short liquidation** totals for a coin over the last
**5 minutes, 1 hour, 4 hours, and 24 hours**, so a user or the AI can ask
“how much BTC was liquidated in the last 24 hours?”

## Behavior

- `GET /api/v1/market/liquidations?symbol=BTCUSDT`
  - `exchange=all` (default) sums **Binance USD-M** and **Bybit linear perpetual**
  - `exchange=binance` or `bybit` for one venue
- Each window includes:
  - `longNotional` / `shortNotional` / `totalNotional` (USDT)
  - `count`
  - `biggest` (side, price, qty, notional, venue, time)
  - `complete` — false until **this coin on this venue** has been tracked for the full window
    (`collectingSince`). Combined `all` uses the **latest** venue start so a newly
    subscribed Bybit coin cannot inherit 24h-complete from a long-running server.
- Streams (background, always on):
  - Binance: `wss://fstream.binance.com/ws/!forceOrder@arr` (all USD-M symbols). At most the **largest** liquidation per symbol per ~1s.
  - Bybit: `wss://stream.bybit.com/v5/public/linear` topic `allLiquidation.{symbol}`. Seed majors + top linear USDT by 24h turnover; querying a symbol also subscribes it.
- Side meaning: **long** = long positions were force-closed; **short** = shorts were force-closed.
- In-memory rolling 24h only. Restart clears history.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation.go` |
| Adapters | `adapter/binance/liqhub.go`, `adapter/bybit/liqhub.go` |
| Service | `GetLiquidations` |
| HTTP | `GET /api/v1/market/liquidations` |
| MCP / AI | `get_liquidations` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/adapter/binance/ ./internal/adapter/bybit/ ./internal/service/market/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT"
curl "http://localhost:8080/api/v1/market/liquidations?symbol=BTCUSDT&exchange=binance"
```

## Limits

- Not Coinbase. Not COIN-M / inverse.
- 24h is complete only after that coin has been watched for 24h on the requested venue(s). A newly tracked coin stays `complete=false` even if the server is older.
- Binance public feed is a 1s largest-hit sample, not every fill.
