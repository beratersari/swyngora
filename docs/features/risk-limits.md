# Feature: Paper portfolio risk limits

## Problem / goal

Emotional trading — especially leveraged margin — can blow up an account. Users should be able to set **their own** optional brakes: stop new risk when the day is already down X%, or stop adding to a coin that is already too large. Limits must **never force-close** what they already hold. They can raise, lower, or remove the rules anytime.

## Behavior

| Rule | Default | When it fires | What is blocked | What is never touched |
|------|---------|---------------|-----------------|------------------------|
| `maxDailyLossPct` | off | Today’s MTM P&L ≤ −cap vs **UTC day baseline** (first `GET`/`PUT`/`guard` of that `YYYY-MM-DD`, not a midnight snapshot) | New spot **buys**, new pending **limit_buy** / **bracket** entries, new **margin long/short** (market or limit). Resting `limit_buy` fills are **not** re-checked. | Sells, stops, OCO exits, closes, cancels, existing size |
| `maxAssetWeightPct` | off | A new buy/open would make that coin **> cap** of current equity (spot value + open margin notional) | New buys/opens **of that coin only** | Other coins, sells, existing overweight (drift is allowed) |
| `maxExposurePct` | off | A new buy/open would make **all coins together** more than the cap as a percent of equity (spot value + open margin notional). Above 100 allows margin gross, up to 1000 | Any new spot buy or margin open | Sells, closes, existing size that already drifted over the cap |

- Limits are stored **per paper book**. `GET`/`PUT`/`DELETE` and order-time guards pass the book id into `View` so they keep working after a second book exists. Extra books can have their own rules.
- No background worker. Rules are checked at order time.
- Recurring DCA buys go through `PlaceOrder`. A risk 403 is stored on the run as generic **`order failed`** (not the full forbidden text); the plan stays active and that period is not retried.
- Basket rebalance sells still run; buys that violate a rule are skipped (same gate). Preview does not simulate risk blocks.
- `PUT` **replaces all three fields** — omit/`null` a field to disable that rule (not a patch merge).
- Clearing limits (`DELETE`) then `PUT` the same UTC day **rebases** start-of-day equity to current equity.

## API (frontend settings screen)

`GET /api/v1/portfolio/risk-limits` returns everything the screen needs in one shot:

```json
{
  "clientId": "…",
  "limits": { "maxDailyLossPct": 5, "maxAssetWeightPct": 30, "maxExposurePct": 60, "updatedAt": "…" },
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

## What-if (not a limit)

`POST /api/v1/portfolio/stress` with `moves: [{asset, movePct}]` reprices the open book before anything happens. BTC −10 and ETH −15 return equity, total P&L, and margin equity before and after. A margin row that crosses its liquidation price loses the margin locked on it. The portfolio screen shows this under **If the market moves**. Nothing is saved or traded. MCP `stress_portfolio` takes `moves` as `BTC:-10,ETH:-15`.

## Limitations

- Saved limits (caps + UTC day baseline) are included when a paper book is exported and restored (`docs/features/user-data-export.md`).
- Daily window is **UTC calendar date**; baseline is first observation that day, not a 00:00 mark-to-market (can add `timeZone` / midnight snapshot later).
- Informational paper trading — not financial advice, not real money.
