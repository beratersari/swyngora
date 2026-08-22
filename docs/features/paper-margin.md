# Feature: Paper margin trading (isolated & cross)

## Problem / goal

Users want leveraged long/short paper trading without real money: market and limit opens, leverage 1x–10x, **isolated or cross** margin mode, correct margin / available balance / PnL / liquidation, auto liquidation, partial closes, add/remove margin (isolated), and stop-loss / take-profit.

## Behavior

### Margin modes (account-wide)

| Mode | Meaning |
|------|---------|
| `isolated` (default) | Only margin assigned to a position backs that position. Add/remove margin allowed. |
| `cross` | Wallet equity is shared; unrealized PnL counts toward equity / liquidation, not spendable cash; liq uses shared equity. |

`PUT /api/v1/portfolio/margin/mode` changes mode. **Blocked** while any open margin position **or** pending margin limit order exists.

### Open
- **Side:** `long` or `short`; **leverage:** 1–10; **type:** `market` or `limit`
- **Idempotency:** same `Idempotency-Key` / `idempotencyKey` as spot — retries of open or close do not create a second position or close trade
- **Initial margin:** `qty * slippedFill / leverage` plus the open taker fee, debited from available cash
- **Fill price:** last plus adverse slippage (long pays more / short receives less); entry stores the fee-inclusive effective price
- **Limit:** reserves required margin **and** worst-case open fee until fill, cancel, or reject (released on cancel)

### Liquidation (maintenance 0.5% of **entry** notional)
- **Isolated:** assigned-margin formula, interest-adjusted (`LiquidationPriceWithDebt`). Long buffer is `margin − maint − debtInterest` (may be **negative** when interest exceeds free buffer → liquidation price rises **above** entry so insolvent longs still liquidate). Short divides by coin debt including interest. Without extra margin/interest this is `entry × (1 − 1/lev + mmr)` long and `entry × (1 + 1/lev − mmr)` short.
- **Cross:** per-position display liq holds other UPNL fixed so this symbol’s mark would drive account equity to total maint. Execution is **account-level**: if equity is under total maint, close the worst position (most negative UPNL) — partial qty when quote debt is tiny (e.g. 1x / no borrow); leveraged longs with quote debt typically **full-close** that position. After each close, re-read cash, sizes, and marks. Debt+qty CAS and deterministic full-close trade ids prevent duplicate records.
- Worker auto-closes when mark crosses liquidation (reason `liquidation`). Last price is used (no separate mark/index).
- **Isolated close / liq:** a losing close cannot consume cash that was never assigned as margin (wallet delta floored at 0). Unassigned cash stays in the book.
- **Cross close / liq:** cash is floored at 0 (account bankruptcy). Unrealized PnL is not spendable for spot or new margin opens.
- **Withdraw (cross):** rejected if post-withdraw equity would fall below total maintenance.
- **Interest worker:** isolated positions use isolated `ShouldLiquidate` after accrue. **Cross** positions never use isolated liq on the interest path — they go through account-level `liquidateCrossIfUnderMaint` (same as maintenance).
- Liq/SL/TP display writes (`UpdateMarginPositionMeta`) do **not** rewrite debt columns, so a stale in-memory row cannot rewind `AccrueInterestCAS`.

### Add / remove margin (isolated only)
- `POST .../positions/{id}/margin` with `delta` (+ add from cash, − return to cash)
- Cannot go below initial margin for remaining size
- Liquidation price recalculated after adjust
- After a successful debit/credit the service reloads the position **by book id**. It does not re-run actor `requireAccess` on the UUID (that failed once a second book existed and a client retry double-charged). Same for repay.

### Borrowing & interest
- **Long:** borrowed **cash** = notional − margin (`debtAsset=quote`); principal and interest shown separately
- **Short:** borrowed **base coins** = quantity (`debtAsset=base`)
- **Interest (persistent background task):** `MarginInterestWorker` (`MARGIN_INTEREST_INTERVAL`, default `1m`)
  - **Catch-up after downtime:** O(1) formula `principal × rate × floor(hours offline)` — does **not** walk each missed hour
  - **Full debt-snapshot CAS** (`debtPrincipal` + `debtInterest` + `lastInterestAt`): two workers cannot apply the same window; restarts do not reprocess a completed period; interest never accrues onto debt that was already repaid or reduced by a concurrent close
  - **Clock skew:** if wall clock moves backward, interest is neither removed nor re-added for past periods
  - **Paid debt:** when `debtPrincipal` is 0, accrual stops (no CAS write)
  - **Same operation after interest:** recompute liquidation price from a **fresh re-read**; if mark is already past liq, **liquidate once**
  - **Single close under concurrency:** liquidation, user close, and repay share debt+quantity CAS; an already-closed position is a no-op for liquidators (no second close trade, no double cash/debt adjustment). Repay/close paths **accrue interest only** — they never nest a second liquidation close
  - **Restart-safe liquidation:** cash + position close + trade commit in **one transaction**. Forced closes use a **deterministic trade id** (`margin-liq:{positionId}`) and a unique index so a second worker/restart cannot credit cash twice or insert two close records. If the process dies mid-tx, SQLite rolls back and the next maintenance tick finishes liquidation; if it dies after commit, the next tick sees `status=closed` / existing trade and is a no-op
- **Borrow limit:** total debt notional ≤ `startingBalance * 9` (10x − own capital)

### Partial close
- Releases **proportional** margin and **proportional debt** (principal + interest); realizes partial PnL; liq recalculated
- Uses the same debt-snapshot CAS as interest/repay and **retries** on conflict so concurrent interest cannot inflate remaining principal/interest incorrectly

### Repay without close
- `POST .../repay` with `amount` in debt units (quote cash for long, coins for short)
- Pays **interest first**, then principal
- Short repay buys coins at mark from available cash
- Accrues under CAS first, then writes repay with expected debt snapshot; on conflict with the interest worker, **retries** with a fresh read so payments are not lost

### Stop-loss / take-profit
- Optional on open or `PUT .../brackets`; worker closes at market when hit

### Equity vs close cash (read before building UI)

Snapshot `equity = cashBalance + spotPositionsValue + marginLocked + marginUnrealizedPnL`.

That figure **does not subtract** outstanding margin debt. Cross liquidation health also uses `cash + Σmargin + ΣUPNL` and **ignores** spot inventory, quote principal, and accrued interest.

**Long close cash** (implemented and tested): `marginRelease + realized − (principalShare + interestShare)`. Opening a long only debits IM; closing still **repays quote principal from cash**. Wallet move is therefore not `IM + UPNL`. Shorts settle without that principal drain (`marginRelease + realized − interest×mark`). Preview close cash with that formula — do not tell users they receive UPNL as cash.

### Paper concurrency
Cash/position read-modify-write is **serialized per `clientId`** in the portfolio service (plus store write mutex and margin debt CAS). The filler may fetch tickers for different symbols in parallel; mutations for the same client wait on one lock.

## API

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/api/v1/portfolio/margin/mode` | Set `isolated` or `cross` |
| `POST` | `/api/v1/portfolio/margin/orders` | Market open or limit rest |
| `GET` | `/api/v1/portfolio/margin/orders` | List orders |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel limit (releases reserve) |
| `GET` | `/api/v1/portfolio/margin/positions` | Open positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | One position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full/partial close |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/margin` | Add/remove isolated margin |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/repay` | Pay interest then principal |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Set/clear SL/TP |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |

Snapshot: `marginMode`, `reservedMargin`, `marginLocked`, `marginUnrealizedPnL`, `marginEquity`, `marginPositions`.

## Web UI (`frontend/`)

Paper margin desk on `/portfolio`:

- Mode toggle (isolated / cross) on the margin ticket
- Open market/limit long/short with leverage and optional SL/TP
- Summary strip shows margin mode, locked margin, margin uPnL
- **Margin positions** tab: entry, mark, liq, debt, close

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/margin.go` |
| Store | `backend/internal/adapter/portfoliostore/margin.go` |
| Service | `backend/internal/service/portfolio/margin.go` |
| Worker | `ProcessMarginMaintenance` via portfolio filler |
| HTTP | `backend/internal/transport/http/handler/portfolio_margin.go` |
| Web | `frontend/src/components/organisms/PaperMarginForm`, `PortfolioMarginPositionsTable`, `pages/PortfolioPage` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ -run Margin -count=1
```

## Limitations

- No funding rates (taker fee + slippage do apply; same rates as spot)
- MMR is a flat 0.5% of **entry** notional (not mark, not tiered)
- Cross health ignores spot collateral and outstanding debt/interest; View.Equity includes spot so it is **not** margin level
- Cross liquidation of under-maint account closes open positions (paper simplification)
- Informational simulation — not real money / not financial advice
