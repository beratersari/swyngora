# Feature: Paper allocation baskets (manual rebalance)

## Problem / goal

Investors think in mixes (“50% BTC, 30% ETH, 20% USDT”). Prices drift. Users want to **see** drift and **optionally** click rebalance — never be forced back to target if they like the new mix (e.g. BTC ran to 65%).

## Behavior

1. Save one or more **named baskets** (max 10 per client, max 20 sleeves). Weights must sum to **100**.
2. Cash sleeve uses the portfolio currency (`USDT` by default). Coins trade as `{ASSET}{currency}` on the target exchange (default binance).
3. **GET** a basket to see live `actualPct` vs `targetPct`. Drift is allowed indefinitely.
4. **Preview** lists proposed market sells (overweight + coins not in the basket) then buys (underweight). No trades.
5. **POST rebalance** is the only path that trades — sells first, then buys at last price, using available (unreserved) cash/qty. Tiny deltas under 1 USDT are skipped.
6. Changing the basket name/targets **does not trade**. There is **no worker** and no auto-rebalance.

Spot equity only (`cash + spot marks` in the portfolio quote). Margin positions are ignored. Out-of-basket coins are treated as 0% target and **sold on rebalance** (shown in preview as `reason=not_in_basket`).

Informational simulation — not financial advice, not real money.

## API

| Method | Path | Trades? |
|--------|------|---------|
| `POST` | `/api/v1/portfolio/baskets` | No |
| `GET` | `/api/v1/portfolio/baskets` | No |
| `GET` | `/api/v1/portfolio/baskets/{id}` | No (live drift + preview legs) |
| `PATCH` | `/api/v1/portfolio/baskets/{id}` | No |
| `DELETE` | `/api/v1/portfolio/baskets/{id}` | No |
| `GET` | `/api/v1/portfolio/baskets/{id}/preview` | No |
| `POST` | `/api/v1/portfolio/baskets/{id}/rebalance` | **Yes** (user click only) |

Example create:

```json
{
  "name": "Core 50/30/20",
  "targets": [
    { "asset": "BTC", "weightPct": 50 },
    { "asset": "ETH", "weightPct": 30 },
    { "asset": "USDT", "weightPct": 20 }
  ]
}
```

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/allocation.go` |
| Store | `backend/internal/adapter/portfoliostore` (`allocation_baskets`, `allocation_targets`) |
| Service | `backend/internal/service/portfolio/allocation.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio_allocation.go` |
| MCP | `create_portfolio_basket`, `preview_portfolio_rebalance`, `rebalance_portfolio_basket`, … |

## How to test

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/transport/http/handler/ -run 'Allocation|PlanRebalance' -count=1
```

## Limitations

- Paper market orders only; no limit rebalance, no fees.
- Only pairs quoted in the portfolio currency count toward the mix.
- Reserved open-order size cannot be sold until released.
