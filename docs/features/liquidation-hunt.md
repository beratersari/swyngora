# Hypothetical liquidation hunt

## Goal

Answer a **what-if** only: if a desk walked **this venue’s spot book** to push
price and force futures liquidations, where is the main long/short pressure,
how much visible spot buying or selling that takes, and would the tour
approximately make or lose money?

This is **not** evidence that Binance or Bybit do this. It is not financial
advice.

## Behavior

`GET /api/v1/market/liquidation-hunt?symbol=BTCUSDT`

- `exchange=all` (default) returns **Binance USD-M and Bybit linear separately**.
  They are never averaged. `exchange=binance` or `bybit` for one venue.
- **Up hunt:** buy spot → lift price → estimated shorts liquidate → they must
  buy to cover. Then unwind the spot (old bids vs assumed cascade exit at the
  target).
- **Down hunt:** sell spot → push price down → estimated longs liquidate →
  buy back cheaper if cascade flow appears.

Each venue report includes:

| Field | Meaning |
|---|---|
| `upPressure` / `downPressure` | Estimated liquidation bands (leverage mix × blended OI share) plus 24h observed clusters |
| `upHunt` / `downHunt` | Primary target (best estimated liq vs spot cost), spot walk, and P&L |
| `spot.notional` | Visible spot size to walk last price to the target |
| `bookOnlyPnl` | Unwind the same size on the **current** opposite side (usually a loss) |
| `netWithCascade` | Part of estimated liquidations becomes exit flow at the target |
| `houseEdge` | `profit` / `loss` / `unreachable` from `netWithCascade` |
| `efficiency` | estimated liquidated notional ÷ spot notional |

### Assumptions (also returned on `assumptions`)

- Isolated liquidation distance ≈ `1/leverage − 0.4%` maintenance, entries at
  the current price.
- Default leverage mix: 5× 15%, 10× 25%, 25× 30%, 50× 18%, 75× 7%, 100× 4%,
  125× 1%. Shifted toward high leverage on the funding-paying side.
- Published long/short is **account count**. Position-share proxy is blended
  60% toward 50/50.
- `liquidationTake` = 0.5% of estimated liquidated notional (insurance-fund-like
  stand-in, not a published fee).
- Spot taker 0.10% each way. Cascade fill rate 50% of estimated liq notional.
- Mark price is a **multi-venue index**. Walking one spot book may not move
  mark 1:1. Hidden liquidity is ignored.

## Where the code lives

| Layer | Path |
|---|---|
| Domain | `backend/internal/domain/liquidation_hunt.go`, `orderbook_reach.go` |
| Service | `backend/internal/service/market/hunt.go` |
| HTTP | `GET /api/v1/market/liquidation-hunt` |
| MCP / AI | `estimate_liquidation_hunt` |

## How to verify

```bash
cd backend && go test ./internal/domain/ ./internal/service/market/ ./internal/transport/http/handler/
curl "http://localhost:8080/api/v1/market/liquidation-hunt?symbol=BTCUSDT"
```

Read `venues[].upHunt` and `venues[].downHunt`. Treat `houseEdge` as a model
output, not a claim about any exchange.
