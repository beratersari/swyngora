# Order-book liquidity flow (traded vs pulled vs added)

## Goal

An 8M bid at 65500 that is 2M five minutes later is **not** automatically
6M of selling. Some of that size was **traded** (market sells hit the
bid). The rest was **pulled** (cancelled). The same split applies to
asks. Over the window, also show how much liquidity was **added**.

## Behavior

`GET /api/v1/market/orderbook/liquidity-flow?symbol=BTCUSDT`

- `exchange=all` (default): **Binance**, **Coinbase**, and **Bybit**
  separately plus `combined`
- Time range: `window=5m|15m|1h` (default `5m`) **or** `startTime` +
  `endTime` (RFC3339 or unix ms). Max **1 hour**
- Optional `minPrice` / `maxPrice` clip the ladder (e.g. `65400`–`65600`)
- `bids` / `asks`: `fromNotional`, `toNotional`, `added`, `pulled`,
  `traded`, `read` (`mostly_traded` | `mostly_pulled` | `mostly_added` |
  `mixed` | `flat`)
- Each `levels[]` row is one price: remaining size, how much was added /
  pulled / traded
- `series[]` is consecutive stored samples (usually 1 minute): bid and
  ask added / pulled / traded in that interval

Attribution (not L3):

```
to = from + added − pulled − traded
```

Aggressive **sells** consume **bids**. Aggressive **buys** consume
**asks**. Size that disappeared without a matching print is **pulled**.
Prints larger than the shrink are treated as **added then eaten**.

Sources: stored **spot** books (1-minute samples of the top grouped
levels) plus **spot** prints (Binance `aggTrades`; Bybit / Coinbase
recent public trades). Need two books in the window. Informational
only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/book_liquidity.go` |
| Service | `backend/internal/service/market/bookliquidity.go` |
| HTTP | `GET /api/v1/market/orderbook/liquidity-flow` |
| MCP / AI | `get_orderbook_liquidity_flow` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/... -run BookLiquidity
curl "http://localhost:8080/api/v1/market/orderbook/liquidity-flow?symbol=BTCUSDT&window=5m"
curl "http://localhost:8080/api/v1/market/orderbook/liquidity-flow?symbol=BTCUSDT&minPrice=65400&maxPrice=65600"
```
