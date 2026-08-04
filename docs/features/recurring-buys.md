# Feature: Paper portfolio recurring buys (DCA)

## Problem / goal

Users want scheduled paper buys of a coin (daily, weekly, or monthly) without placing each market order by hand. Missed periods must not cascade into a flood of catch-up buys, and restarts/concurrent workers must not double-execute the same period.

## Behavior

1. **Create** a plan: coin (`exchange` + `symbol`), cash `amount`, `frequency` (`daily` | `weekly` | `monthly`), optional `startAt`.
2. **Pause / resume / delete** at any time.
3. When due, the worker buys at **last market price** with `quantity = amount / price`.
4. **Insufficient cash** (or market unavailable): that run is stored as **failed**; the plan stays **active** and advances to the next period.
5. **Idempotency:** `UNIQUE(plan_id, period_key)` on run rows — only one claim per calendar period.
6. **Catch-up:** if the server was down across multiple periods, only the **latest** due slot runs once (intermediate slots are skipped).

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio/recurring-buys` | Create plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume |
| `DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Delete |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Run history |

Requires an existing paper portfolio for the same `clientId`. Max 20 plans per client.

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/recurring_buy.go` |
| Store | `backend/internal/adapter/portfoliostore/sqlite.go` |
| Service | `backend/internal/service/portfolio/recurring_buy.go` |
| Worker | `backend/internal/service/portfolio/recurring_worker.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio_recurring.go` |
| MCP / AI | Go MCP tools + `ai/src/swyngora_ai/tools/market_http.py` |

## How to test

```bash
cd backend
go test ./internal/domain/ -run Recurring -count=1
go test ./internal/service/portfolio/ -run Recurring -count=1
```

Manual: create portfolio → create plan with `startAt` in the past → wait for worker (or call process via tests) → check runs and cash.

## Config

| Env | Default | Meaning |
|-----|---------|---------|
| `RECURRING_BUY_INTERVAL` | `30s` | Worker poll interval |
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` | Plans + runs stored with portfolio data |

## Limitations / follow-ups

- Paper trading only; not real money.
- Market orders only (cash notional); no limit-price recurring buys yet.
- Plans are not included in user data export/import yet.
- UI for product web/mobile not in this change.
