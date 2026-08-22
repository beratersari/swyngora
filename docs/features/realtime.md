# Realtime prices and paper portfolio updates

## Goal

Stop polling REST for live numbers. One WebSocket lets the client subscribe to **selected coins** and the **selected paper portfolio**. After a drop, reconnect and resubscribe; the server snapshots current state. Portfolio events are only sent if the `clientId` can view that book (owner, trader, or shared viewer).

## Connect

| Item | Value |
|------|--------|
| Upgrade | `GET /api/v1/ws?clientId=<id>` |
| Protocol info | `GET /api/v1/realtime` |
| Auth | Same as REST (`Authorization` / `X-API-Key`). Browsers: `?token=` |
| Max price symbols | 100 per connection |
| Price pump | `REALTIME_PRICE_INTERVAL` (default 5s; uses ticker cache) |

## Client → server

```json
{ "type": "subscribe_prices", "symbols": [{ "exchange": "binance", "symbol": "BTCUSDT" }] }
{ "type": "unsubscribe_prices", "symbols": [{ "exchange": "binance", "symbol": "BTCUSDT" }] }
{ "type": "subscribe_portfolio", "portfolioId": "<book-id>" }
{ "type": "unsubscribe_portfolio" }
{ "type": "ping" }
```

Empty `symbols` on unsubscribe_prices drops all price subscriptions.

## Server → client

- `hello` — protocol version + clientId
- `ack` — subscription confirmed
- `price` — lastPrice / 24h fields for one pair, plus `halted` when the last print is a delist halt (same meaning as REST `ticker.halted`)
- `portfolio` — `reason` is `snapshot` \| `order_placed` \| `order_amended` \| `order_cancelled` \| `order_filled` \| `order_updated` \| `cash`; includes `portfolio` snapshot plus optional `order` / `trade` / `orders`
- `error` — invalid request or access revoked
- `pong`

On `subscribe_prices` / `subscribe_portfolio` the server immediately sends the current ticker(s) and book snapshot + open orders so a reconnect continues without a REST refetch.

## Access

`subscribe_portfolio` uses the same rules as `GET /api/v1/portfolio` (viewer or above). Shared traders/viewers get events; if the share is revoked, the socket receives `error` and the subscription is dropped.

## Frontend

`frontend/src/libs/realtime/` opens one connection, refcounts subscriptions, patches RTK Query caches, and falls back to HTTP polling while disconnected. Vite proxies `/api` with `ws: true`.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/realtime.go` |
| Hub | `backend/internal/service/realtime/` |
| Transport | `backend/internal/transport/http/handler/realtime.go` |
| MCP | `realtime_stream_info` |
| Web client | `frontend/src/libs/realtime/` |

## Verify

```bash
cd backend && go test ./internal/service/realtime/ ./internal/transport/http/handler/ -count=1
# after server start:
# wscat -c "ws://127.0.0.1:8080/api/v1/ws?clientId=demo"
```

## Limits

- Candle/indicator series still poll (not streamed).
- Price freshness follows `TICKER_CACHE_TTL` (default 15s).
- One selected portfolio per connection.
