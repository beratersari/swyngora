# Paper trading / virtual portfolio

## Goal

Simulated portfolios with starting cash, market buy/sell at last price, **pending limit/stop orders**, open positions, realized/unrealized P&L, and trade history. **Not real money.** Data is stored in SQLite and survives restarts.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio` | Create portfolio (`startingBalance`, optional `currency`) |
| `GET` | `/api/v1/portfolio` | Snapshot: cash, equity, positions, P&L |
| `POST` | `/api/v1/portfolio/orders` | Market or pending order (see below) |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (`status` default `open`; also `filled` / `canceled` / `rejected` / `all`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel an **open** pending order |
| `GET` | `/api/v1/portfolio/trades` | Trade history (`limit`, `offset`) |

Tenancy uses the same `clientId` / `X-Client-Id` model as watchlists (one portfolio per client).

### Order types (`POST /api/v1/portfolio/orders`)

| `type` | Required fields | Fill condition (last price) | Side |
|--------|-----------------|------------------------------|------|
| `market` (default) | `side`, `quantity` | Immediate at last price | buy / sell |
| `limit_buy` | `quantity`, `triggerPrice` | last ≤ trigger | buy |
| `limit_sell` | `quantity`, `triggerPrice` | last ≥ trigger | sell |
| `stop_loss` | `quantity`, `triggerPrice` | last ≤ trigger | sell |

Pending fills use the **current last price** when the condition is met (paper simulation). Cash and position are checked **at fill time**; if insufficient, the order is marked `rejected` and never fills later.

## Behavior

- **Buy:** debit cash at fill price; increase position; average cost updated.
- **Sell:** credit cash; reduce position; realize `(price - avgCost) * qty`.
- **Unrealized P&L:** mark open positions to current last price.
- **Equity:** cash + market value of positions.
- **Pending orders:** stored in SQLite; background filler evaluates on `PORTFOLIO_ORDER_CHECK_INTERVAL` (and once on process start).
- **Idempotency:** fill/cancel use `status = open` predicates so the same order cannot fill twice and canceled orders never execute.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go` |
| Store | `backend/internal/adapter/portfoliostore` |
| Service | `backend/internal/service/portfolio` |
| Filler | `backend/internal/service/portfolio/filler.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio.go` |
| MCP | `create_portfolio`, `get_portfolio`, `place_portfolio_order`, `place_portfolio_pending_order`, `list_portfolio_orders`, `cancel_portfolio_order`, `list_portfolio_trades` |

## Config

| Env | Default |
|-----|---------|
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ -run Portfolio -count=1
```