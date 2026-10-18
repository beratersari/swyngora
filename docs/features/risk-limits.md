# Feature: Paper portfolio risk limits

## Problem / goal

Emotional trading — especially leveraged margin — can blow up an account. Users should be able to set **their own** optional brakes: stop new risk when the day is already down X%, or stop adding to a coin that is already too large. Limits must **never force-close** what they already hold. They can raise, lower, or remove the rules anytime. Closing positions when today's trading loss passes a chosen percent is a separate switch: `docs/features/emergency-stop.md`.

## Behavior

| Rule | Default | When it fires | What is blocked | What is never touched |
|------|---------|---------------|-----------------|------------------------|
| `maxDailyLossPct` | off | Today’s trading P&L ≤ −cap vs **UTC day baseline** (first `GET`/`PUT`/`guard` of that `YYYY-MM-DD`, not a midnight snapshot). Deposits, withdrawals, and transfers are not part of that P&L | New spot **buys**, new pending **limit_buy** / **bracket** entries, new **margin long/short** (market or limit). Resting `limit_buy` fills are **not** re-checked. | Sells, stops, OCO exits, closes, cancels, existing size |
| `maxAssetWeightPct` | off | A new buy/open would make that coin **> cap** of current equity (spot value + open margin notional) | New buys/opens **of that coin only** | Other coins, sells, existing overweight (drift is allowed) |
| `maxExposurePct` | off | A new buy/open would make **all coins together** more than the cap as a percent of equity (spot value + open margin notional). Above 100 allows margin gross, up to 1000 | Any new spot buy or margin open | Sells, closes, existing size that already drifted over the cap |

- Limits are stored **per paper book**. `GET`/`PUT`/`DELETE` and order-time guards pass the book id into `View` so they keep working after a second book exists. Extra books can have their own rules.
- No background worker. Rules are checked at order time.
- Recurring DCA buys go through `PlaceOrder`. A risk 403 is stored on the run as generic **`order failed`** (not the full forbidden text); the plan stays active and that period is not retried.
- Basket rebalance sells still run; buys that violate a rule are skipped (same gate). The basket preview does not simulate risk blocks.
- `POST /api/v1/portfolio/orders/preview` does report whether this one order would be blocked. It does not place the order. A sell still shows today's trading loss when the daily limit is already hit, and it is not blocked. A `riskPercent` sizes the order from the stop. That percent is how much of the book the stop may lose. It is not the daily-loss limit.
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
| MCP | `get_portfolio_risk_limits`, `set_portfolio_risk_limits`, `clear_portfolio_risk_limits`, `get_portfolio_correlation`, `preview_portfolio_order` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/transport/http/handler/ -run 'Risk|Stress|Replay' -count=1
```

## How the coins move together

`GET /api/v1/portfolio/correlation` reads the open spot and margin coins over the last day of 5-minute percent changes. `together` is the size-weighted average correlation. `pairs` lists each coin pair with the strongest absolute correlation first, whether that link is positive or negative. A pair with no correlation stays last. `swingPct` is the size of a day built from those moves. `apartPct` is the same mix if the coins had moved independently. A short is the opposite of a long, so two coins that rise together can still hedge when one is short. A coin without enough shared minutes is named in `missing` and is not treated as zero. One coin has a swing and no pair.

Optional `addSymbol`, `addSide` (`buy`, `sell`, `long`, or `short`), and `addNotional` return `add` for the book after that extra line. Nothing is placed. The order preview also returns this read as `correlation`, including `add` for the order on the ticket. A failed read leaves `correlation` off the preview and still returns the price.

The portfolio screen shows the open book under the risk limits. The ticket says whether the order raises or lowers the day swing. This read does not block an order.

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/transport/http/handler/ -run 'PortfolioCorrelation|BookCorrelation|PreviewOrder_Opposite|PortfolioHTTP_Correlation' -count=1
```

## What-if (not a limit)

`POST /api/v1/portfolio/stress` reprices the open book before anything happens. Nothing is saved or traded.

Typed percents use `moves: [{asset, movePct}]`. BTC −10 and ETH −15 return equity, total P&L, and margin equity before and after. A margin row that crosses its liquidation price loses the margin locked on it.

A past window uses `from` and `to` instead of `moves`. Send one of those, not both. `YYYY-MM-DD` is a UTC day and the end day is included. The start price is the close entering the first day. The end price is the close of the last day. An RFC3339 pair uses the last close at or before each instant. An end in the future is cut back to now. A window of at least 24 hours uses daily closes. A shorter window uses hourly closes. The window has to be at least an hour and at most 10 years.

Each open position keeps its own venue's change. The coins are listed by the size of that price change, largest up or down first. Equal sizes stay in name order. Spot weight is market value. Margin weight is notional (quantity times mark). When the same coin is open on two venues, each venue is repriced on its own, and `movePct` on the coin is the value-weighted average of the venues that priced. A coin or venue with no close at either end stays unchanged and is named (`no_start`, `no_end`, `same_close`, `out_of_range`, `no_history`, `fetch_failed`). Cash does not change. Equities read the venue's latest daily or hourly window (about two years of daily bars, about a month of hourly bars). A close outside that window is named and left unchanged.

The portfolio screen shows this under **If the market moves**, with **Type a move** and **Replay a period**. The equity line shows the money change and that change as a percent of current equity. A book at zero or below keeps the money change and leaves the percent off. A replay also draws the book at today's size on each close in the window, and names the lowest point. A margin position that liquidates on that path stays liquidated afterward. MCP and the paper assistant tool `stress_portfolio` take `moves` as `BTC:-10,ETH:-15`, or `from` / `to` (`period_from` / `period_to` in Python).

## Limitations

- Saved limits (caps + UTC day baseline) are included when a paper book is exported and restored (`docs/features/user-data-export.md`).
- Daily window is **UTC calendar date**; baseline is first observation that day, not a 00:00 mark-to-market (can add `timeZone` / midnight snapshot later).
- Informational paper trading — not financial advice, not real money.
