# Paper trading / virtual portfolio

## Goal

Simulated portfolios with starting cash, market buy/sell at last price, **pending limit/stop orders with reservations and partial fills**, open positions, realized/unrealized P&L, and trade history. **Not real money.** Data is stored in SQLite and survives restarts.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio` | Create portfolio (`startingBalance`, optional `currency`) |
| `GET` | `/api/v1/portfolio` | Snapshot: cash, reserved/available cash, positions, P&L |
| `POST` | `/api/v1/portfolio/orders` | Market or pending order (see below) |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (`status` default `open`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel open pending order (releases unused reservation) |
| `GET` | `/api/v1/portfolio/trades` | Trade history (`limit`, `offset`); pending fills include `pendingOrderId` |
| `POST` | `/api/v1/portfolio/recurring-buys` | Create recurring buy (DCA) plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Delete plan (+ run history) |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Execution history |

Tenancy uses the same `clientId` / `X-Client-Id` model as watchlists (one portfolio per client).

### Recurring buys (DCA)

| Field | Description |
|-------|-------------|
| `symbol` + `exchange` | Coin pair to buy |
| `amount` | Cash notional spent each run at last market price (`qty = amount / price`) |
| `frequency` | `daily` \| `weekly` \| `monthly` |
| `startAt` | Optional first run (RFC3339); default now |

**Lifecycle:** create (active) → pause / resume → delete. Failed runs (e.g. insufficient cash) keep the plan active and only that period is recorded as failed.

**Safety:**
- `UNIQUE(plan_id, period_key)` claim so restarts and concurrent workers cannot double-buy the same period.
- Missed windows run **only the latest** due slot (no backlog of intermediate buys).
- Worker interval: `RECURRING_BUY_INTERVAL` (default `30s`).

### Order types (`POST /api/v1/portfolio/orders`)

| `type` | Required fields | Fill condition (last price) | Side | Reservation |
|--------|-----------------|------------------------------|------|-------------|
| `market` (default) | `side`, `quantity` | Immediate at last price | buy / sell | Uses **available** cash/qty only |
| `limit_buy` | `quantity`, `triggerPrice` | last ≤ trigger | buy | Reserves `quantity * triggerPrice` cash |
| `limit_sell` | `quantity`, `triggerPrice` | last ≥ trigger | sell | Reserves `quantity` of position |
| `stop_loss` | `quantity`, `triggerPrice` | last ≤ trigger | sell | Reserves `quantity` of position |

Optional pending fields:
- `timeInForce`: `gtc` (default), `ioc`, or `fok`
- `expiresAt`: RFC3339 timestamp (**GTC only**); when reached the order is canceled and unused reservation is released

## Behavior

### Time-in-force
| Policy | Behavior |
|--------|----------|
| **GTC** | Stays open until filled, user cancel, or `expiresAt`. Partial fills allowed; remainder stays open. |
| **IOC** | On the first try: fill as much as possible if marketable; cancel any remainder (`ioc_remainder`) or cancel with no fill (`ioc_no_fill`). Reservation released. |
| **FOK** | On the first try: fill **entire** remaining size or cancel with **no** fill (`fok_unfilled`). |

### Reservations
- **Buy pending:** locks cash so another market/pending buy cannot spend it.
- **Sell pending:** locks position quantity so another market/pending sell cannot sell it.
- Snapshot fields: `reservedCash`, `availableCash`; positions include `reservedQuantity`, `availableQuantity`.
- **Cancel / reject / expire / IOC remainder / FOK kill:** remaining reservation is released immediately (`cancelReason` explains why).

### Partial fills
- Orders track `quantity` (original), `filledQuantity`, `remainingQuantity`.
- When triggered, the filler may fill only part of the remaining size; the order stays `open` until remaining is zero, then becomes `filled`.
- **Each fill** creates a separate trade history row with `pendingOrderId`.
- Latest fill metadata: `fillTradeId`, `fillPrice`.

### Accounting
- **Buy fill:** debit cash at fill price; increase position; average cost updated.
- **Sell fill:** credit cash; reduce position; realize `(price - avgCost) * qty`.
- **Equity:** cash + market value of positions (reserved cash is still part of cash until filled).

### Durability & safety
- Portfolios, positions, trades, pending orders, remaining size, reservations, and recurring buy plans/runs are in SQLite (`PORTFOLIO_DB_PATH`).
- Fill/cancel/reject use `status = open` predicates so a canceled order never fills and a completed order is not double-filled.
- Background filler runs on `PORTFOLIO_ORDER_CHECK_INTERVAL` and once on process start.
- Recurring buy worker runs on `RECURRING_BUY_INTERVAL` and once on process start.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go`, `recurring_buy.go` |
| Store | `backend/internal/adapter/portfoliostore` |
| Service | `backend/internal/service/portfolio` |
| Filler | `backend/internal/service/portfolio/filler.go` |
| Recurring worker | `backend/internal/service/portfolio/recurring_worker.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio.go`, `portfolio_recurring.go` |
| MCP | `create_portfolio`, `get_portfolio`, `place_portfolio_order`, `place_portfolio_pending_order`, `list_portfolio_orders`, `cancel_portfolio_order`, `list_portfolio_trades`, `create_recurring_buy`, `list_recurring_buys`, `get_recurring_buy`, `pause_recurring_buy`, `resume_recurring_buy`, `delete_recurring_buy`, `list_recurring_buy_runs` |

## Config

| Env | Default |
|-----|---------|
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` |
| `RECURRING_BUY_INTERVAL` | `30s` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ -run 'Portfolio|Recurring' -count=1
```