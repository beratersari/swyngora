# Feature: Paper portfolio risk limits

## Problem / goal

Emotional trading — especially leveraged margin — can blow up an account. Users should be able to set **their own** optional brakes: stop new risk when the day is already down X%, or stop adding to a coin that is already too large. Limits must **never force-close** what they already hold. They can raise, lower, or remove the rules anytime.

## Behavior

| Rule | Default | When it fires | What is blocked | What is never touched |
|------|---------|---------------|-----------------|------------------------|
| `maxDailyLossPct` | off | Today’s MTM P&L ≤ −cap vs **UTC start-of-day equity** | New spot **buys**, new pending **limit_buy** / **bracket** entries, new **margin long/short** (market or limit) | Sells, stops, OCO exits, closes, cancels, existing size |
| `maxAssetWeightPct` | off | A new buy/open would make that coin **> cap** of current equity (spot value + open margin notional) | New buys/opens **of that coin only** | Other coins, sells, existing overweight (drift is allowed) |

- No background worker. Rules are checked at order time.
- Recurring DCA buys go through `PlaceOrder` and fail that run with the same block.
- Basket rebalance sells still run; buys that violate a rule are skipped (same gate).
- Clearing limits (`DELETE`) or sending `null` for a field turns that rule off.

## API (frontend settings screen)

`GET /api/v1/portfolio/risk-limits` returns everything the screen needs in one shot:

```json
{
  "clientId": "…",
  "limits": { "maxDailyLossPct": 5, "maxAssetWeightPct": 30, "updatedAt": "…" },
  "status": {
    "dayKey": "2026-08-07",
    "timezone": "UTC",
    "startOfDayEquity": 10000,
    "equity": 9480,
    "dailyPnl": -520,
    "dailyPnlPct": -5.2,
    "dailyLossLimitHit": true,
    "assets": [{ "asset": "BTC", "value": 6200, "weightPct": 65.4, "atOrOverLimit": true }],
    "canOpenSpotBuy": false,
    "canOpenMargin": false,
    "blockReasons": ["daily loss limit reached (5.20% loss >= 5.00%)"]
  },
  "note": "…"
}
```

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/portfolio/risk-limits` | Load form + live meters |
| `PUT` | `/api/v1/portfolio/risk-limits` | Save both fields (`null` / omit = disable that rule) |
| `DELETE` | `/api/v1/portfolio/risk-limits` | Remove all rules |

Blocked trades return **403** `forbidden` with a human `message` (e.g. `BTC would be 32.10% of the portfolio (limit 30.00%)`).

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/risk.go` |
| Store | `risk_limits` table in `portfoliostore` |
| Service | `backend/internal/service/portfolio/risk.go` (`guardNewRisk` on buy / margin open) |
| HTTP | `backend/internal/transport/http/handler/portfolio_risk.go` |
| MCP | `get_portfolio_risk_limits`, `set_portfolio_risk_limits`, `clear_portfolio_risk_limits` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/transport/http/handler/ -run 'Risk' -count=1
```

## Limitations

- Daily window is **UTC midnight**, not the user’s local timezone (can add `timeZone` later).
- Informational paper trading — not financial advice, not real money.
