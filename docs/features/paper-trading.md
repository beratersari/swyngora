# Paper trading / virtual portfolio

## Goal

Simulated portfolios with starting cash, market buy/sell at last price, open positions, realized/unrealized P&L, and trade history. **Not real money.** Data is stored in SQLite and survives restarts.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio` | Create portfolio (`startingBalance`, optional `currency`) |
| `GET` | `/api/v1/portfolio` | Snapshot: cash, equity, positions, P&L |
| `POST` | `/api/v1/portfolio/orders` | Market `buy` / `sell` (`symbol`, `quantity`, optional `exchange`) |
| `GET` | `/api/v1/portfolio/trades` | Trade history (`limit`, `offset`) |

Tenancy uses the same `clientId` / `X-Client-Id` model as watchlists (one portfolio per client).

## Behavior

- **Buy:** debit cash at last price; increase position; average cost updated.
- **Sell:** credit cash; reduce position; realize `(price - avgCost) * qty`.
- **Unrealized P&L:** mark open positions to current last price.
- **Equity:** cash + market value of positions.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go` |
| Store | `backend/internal/adapter/portfoliostore` |
| Service | `backend/internal/service/portfolio` |
| HTTP | `backend/internal/transport/http/handler/portfolio.go` |
| MCP | `create_portfolio`, `get_portfolio`, `place_portfolio_order`, `list_portfolio_trades` |

## Config

| Env | Default |
|-----|---------|
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ -run Portfolio -count=1
```