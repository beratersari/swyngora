# Feature: Paper portfolio recurring buys (DCA)

## Problem / goal

Users want scheduled paper buys of a coin without placing each market order by hand — every Monday, every 12 hours, or on salary day — with a memorable name. Missed periods must not cascade into a flood of catch-up buys, and restarts/concurrent workers must not double-execute the same period.

## Behavior

1. **Create** a plan: coin (`exchange` + `symbol`), cash `amount`, optional **`name`**, `frequency`, optional `startAt`, optional `timeZone` / `hour` / `minute`, optional `maxPrice`.
2. **Schedules**
   | frequency | Extra fields | Example |
   |-----------|--------------|---------|
   | `daily` | optional `timeZone` + `hour`/`minute` | 50 USDT BTC every day at 09:00 Istanbul |
   | `weekly` | `weekday` (`monday`…`sunday`) + optional local clock | 500 USDT BTC every Monday 09:00 `Europe/Istanbul` |
   | `monthly` | `dayOfMonth` `1–31` (clamped on short months) | 1500 USDT ETH on the 25th (salary day) |
   | `interval` | `intervalHours` `1–168` | 1500 USDT ETH every 12 hours (elapsed UTC, not wall clock) |
3. **Local clock:** IANA `timeZone` (e.g. `Europe/Istanbul`). Empty timezone keeps the previous UTC `startAt` clock. `hour` `0–23` and `minute` `0–59` pin the local time; weekday and month day are interpreted in that zone.
4. **Name** (max 80 chars): e.g. `"Salary Day Buy"`, `"Buy Coins With 30% of My Money"`. Empty → `"BTCUSDT monthly"`. Names are labels only — amount is still a fixed cash notional, not a live % of balance.
5. **Pause / resume / delete** at any time. **PATCH** updates name, amount, schedule, timezone/clock, and/or `maxPrice`.
6. When due, the worker buys **on the book that owns the plan** (Main or a named book) at **last price plus buy slippage** with `quantity = amount / (slippedFill × (1 + fee))` so the cash spend includes the taker fee. Sizing and the market fill use the **same last print** so a later ticker cannot overspend the plan. The plan stores the book id; execution places the order as the owner with that `portfolioId` so a second book does not block Main and UUID books still fill.
7. **`maxPrice`:** optional cap (0 = none). The run is skipped when **last**, **slipped fill**, or **fee-inclusive unit cost** would exceed it. Example: cap 65000 — last 66000 does not buy; last 64000 buys at the current slipped market; last 64980 can still skip if slip+fee pushes the real unit over 65000. A skipped run is **failed** with that reason; the plan stays **active** and advances.
8. **Insufficient cash** (or market unavailable): that run is stored as **failed**; the plan stays **active** and advances to the next period.
9. **Idempotency:** `UNIQUE(plan_id, period_key)` on run rows — only one claim per period.
10. **Catch-up:** if the server was down across multiple periods, only the **latest** due slot runs once (intermediate slots are skipped).

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio/recurring-buys` | Create plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `PATCH` | `/api/v1/portfolio/recurring-buys/{id}` | Update name / amount / schedule / timeZone / hour / minute / maxPrice |
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
go test ./internal/transport/http/handler/ -run RecurringBuy -count=1
```

Manual: create portfolio → create a second book → create a plan with `startAt` in the past on each book → wait for worker (or call process via tests) → check runs and cash on the matching book.

## Config

| Env | Default | Meaning |
|-----|---------|---------|
| `RECURRING_BUY_INTERVAL` | `30s` | Worker poll interval |
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` | Plans + runs stored with portfolio data |

## Limitations / follow-ups

- Paper trading only; not real money.
- Market orders only (cash notional). `maxPrice` skips the period; it does not place a limit order.
- Name does not compute a percent of wallet — use a fixed `amount`.
- Weekday / day-of-month / clock use **UTC** unless `timeZone` is set.
- UI for product web/mobile not in this change.
