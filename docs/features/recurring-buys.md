# Feature: Paper portfolio recurring buys (DCA)

## Problem / goal

Users want scheduled paper buys of a coin without placing each market order by hand — every Monday, every 12 hours, or on salary day — with a memorable name. Missed periods must not cascade into a flood of catch-up buys, and restarts/concurrent workers must not double-execute the same period.

## Behavior

1. **Create** a plan: coin (`exchange` + `symbol`), cash `amount`, optional **`name`**, `frequency`, optional `startAt`.
2. **Schedules**
   | frequency | Extra fields | Example |
   |-----------|--------------|---------|
   | `daily` | — | 50 USDT BTC every day |
   | `weekly` | `weekday` (`monday`…`sunday`, UTC) | 500 USDT BTC every Monday |
   | `monthly` | `dayOfMonth` `1–31` (clamped on short months) | 1500 USDT ETH on the 25th (salary day) |
   | `interval` | `intervalHours` `1–168` | 1500 USDT ETH every 12 hours |
3. **Name** (max 80 chars): e.g. `"Salary Day Buy"`, `"Buy Coins With 30% of My Money"`. Empty → `"BTCUSDT monthly"`. Names are labels only — amount is still a fixed cash notional, not a live % of balance.
4. **Pause / resume / delete** at any time. **PATCH** updates name, amount, and/or schedule.
5. When due, the worker buys at **last price plus buy slippage** with `quantity = amount / (slippedFill × (1 + fee))` so the cash spend includes the taker fee.
6. **Insufficient cash** (or market unavailable): that run is stored as **failed**; the plan stays **active** and advances to the next period.
7. **Idempotency:** `UNIQUE(plan_id, period_key)` on run rows — only one claim per period.
8. **Catch-up:** if the server was down across multiple periods, only the **latest** due slot runs once (intermediate slots are skipped).

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio/recurring-buys` | Create plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `PATCH` | `/api/v1/portfolio/recurring-buys/{id}` | Update name / amount / schedule |
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
- Name does not compute a percent of wallet — use a fixed `amount`.
- Weekday / day-of-month use **UTC**.
- Plans are not included in user data export/import yet.
- UI for product web/mobile not in this change.
