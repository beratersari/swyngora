# Iceberg refill

## Goal

See when liquidity at the **same price** keeps coming back after trades —
the “10k sell, buyers eat it, another 10k appears” pattern on **both**
the buy and sell side. That is different from a one-shot wall.

## Behavior

`GET /api/v1/market/orderbook/icebergs?symbol=BTCUSDT`

A clip is flagged when:

1. Visible size at one price is large enough (default **$25k** notional)
2. That size is **eaten** at the touch (or taker prints hit that price)
3. A similar clip **refills** at the same price
4. This happens **at least twice**

Pulled walls far from the touch (spoof / flicker) are **not** icebergs —
those stay `suspicious` on the live book.

Live book walls also get `behavior=iceberg`, `icebergRefills`, and
`icebergClip` after the same pattern. The 3-second wall sampler keeps
watching after you look at a book.

Optional: `exchange=binance|coinbase|bybit|all`, `minNotional`.

Book-pattern only — not proof of a hidden exchange order. Informational
only — not financial advice.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/iceberg.go` |
| Service | `backend/internal/service/market/icebergs.go` |
| HTTP | `GET /api/v1/market/orderbook/icebergs` |
| MCP / AI | `get_orderbook_icebergs` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ -run Iceberg
curl "http://localhost:8080/api/v1/market/orderbook/icebergs?symbol=BTCUSDT"
```
