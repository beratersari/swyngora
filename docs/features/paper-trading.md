# Paper trading / virtual portfolio

## Goal

Simulated portfolios with starting cash, market buy/sell at last price, **pending limit/stop orders with reservations and partial fills**, open positions, realized/unrealized P&L, and trade history. **Not real money.** Data is stored in SQLite and survives restarts.

Live order/position/cash updates for a selected book (and price ticks for selected coins) go over **`GET /api/v1/ws`** — see [`realtime.md`](realtime.md).

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio` | Create a named book (`startingBalance`, optional `name`, `currency`) |
| `GET` | `/api/v1/portfolios` | List books for the client |
| `PATCH` | `/api/v1/portfolios/{id}` | Rename a book |
| `DELETE` | `/api/v1/portfolios/{id}` | Delete a book and all of its data |
| `GET` | `/api/v1/portfolio` | Snapshot of the selected book (`portfolioId` / `X-Portfolio-Id`) |
| `POST` | `/api/v1/portfolio/deposits` | Add virtual cash (`amount`, optional `note`) |
| `POST` | `/api/v1/portfolio/withdrawals` | Withdraw available cash (`amount`, optional `note`) |
| `POST` | `/api/v1/portfolio/transfers` | Move cash between your own books (`fromPortfolioId`, `toPortfolioId`, `amount`) — owner only |
| `GET` | `/api/v1/portfolio/cash-movements` | Deposit / withdraw / transfer history (newest first; includes opening) |
| `GET` | `/api/v1/portfolio/performance` | Equity series + period P&L (`period=1d\|1w\|1m\|3m`) |
| `GET`/`PUT`/`DELETE` | `/api/v1/portfolio/risk-limits` | Optional daily-loss % and max coin weight % (block new buys/margin only) |
| `POST` | `/api/v1/portfolio/orders` | Market or pending order (see below) |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (`status` default `open`) |
| `GET` | `/api/v1/portfolio/orders/{id}` | One order + last price + amend hints for the edit screen |
| `PATCH` | `/api/v1/portfolio/orders/{id}` | Amend open GTC limit/stop in place (`triggerPrice`, `remainingQuantity`) |
| `POST` | `/api/v1/portfolio/orders/cancel-all` | Cancel all open orders, or one market (`symbol` + optional `exchange`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel open pending order (releases unused reservation) |
| `GET` | `/api/v1/portfolio/trades` | Trade history (`limit`, `offset`); pending fills include `pendingOrderId`; `price` is slipped fill, `fee` is quote taker fee, `lastPrice` is pre-slip last |
| `GET` | `/api/v1/portfolio/trading-costs` | Per-exchange paper taker fee and slippage (`?exchange=` optional) |
| `GET` | `/api/v1/portfolio/lots` | Tax lots (`status=open\|closed\|all`, optional `exchange`/`symbol`) |
| `POST` | `/api/v1/portfolio/recurring-buys` | Create named recurring buy (DCA) plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `PATCH` | `/api/v1/portfolio/recurring-buys/{id}` | Update name / amount / schedule |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Delete plan (+ run history) |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Execution history |
| `POST` | `/api/v1/portfolio/baskets` | Create named allocation basket (target % mix; no trades) |
| `GET` | `/api/v1/portfolio/baskets` | List baskets |
| `GET` | `/api/v1/portfolio/baskets/{id}` | Basket + live actual vs target |
| `PATCH` | `/api/v1/portfolio/baskets/{id}` | Update name/targets (no trades) |
| `DELETE` | `/api/v1/portfolio/baskets/{id}` | Delete basket |
| `GET` | `/api/v1/portfolio/baskets/{id}/preview` | Proposed rebalance legs (no trades) |
| `POST` | `/api/v1/portfolio/baskets/{id}/rebalance` | User-triggered rebalance at last price |
| `POST` | `/api/v1/portfolio/margin/orders` | Margin market/limit long or short (1x–10x) |
| `GET` | `/api/v1/portfolio/margin/orders` | List margin orders |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel margin limit order |
| `GET` | `/api/v1/portfolio/margin/positions` | Open margin positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | Get margin position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full or partial close |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Set/clear stop-loss / take-profit |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |

Tenancy uses the same `clientId` / `X-Client-Id` model as watchlists. Each client may own **up to 20 named paper books**. The first book keeps id = `clientId` and default name `Main` (legacy). Additional books get a UUID id. All action routes (`/orders`, `/deposits`, `/performance`, …) accept optional `portfolioId` query, `X-Portfolio-Id` header, or body field. Omit it when there is exactly one book; if several exist, the API returns 400 until you select one.

### Sharing

The owner can share **one book** with another `clientId`:

| Role | View snapshot / positions / trades / performance | Place & cancel orders | Deposit / withdraw / **transfer** / delete / manage shares |
|------|--------------------------------------------------|------------------------|------------------------------------------------------------|
| **owner** | yes | yes | yes |
| **trader** | yes | yes | no |
| **viewer** | yes | no | no |

| Method | Path | Who |
|--------|------|-----|
| `POST` | `/api/v1/portfolio/shares` | owner `{ granteeClientId, role: viewer\|trader, portfolioId }` |
| `PATCH` | `/api/v1/portfolio/shares` | owner change role |
| `GET` | `/api/v1/portfolio/shares` | owner list outgoing |
| `DELETE` | `/api/v1/portfolio/shares?granteeClientId=` | owner revoke |
| `GET` | `/api/v1/portfolios/shared` | grantee incoming books |

Grantees select the book with `portfolioId` (UUID) or `ownerClientId` + name. Cannot share with yourself. Max 50 shares per book.

MCP: `list_portfolios`, `create_portfolio` (name), `rename_portfolio`, `delete_portfolio`, `share_portfolio`, `update_portfolio_share`, `revoke_portfolio_share`, `list_portfolio_shares`, `list_shared_portfolios`; other portfolio tools take optional `portfolioId`. Telegram: `/portfolio list`, `/portfolio create [balance] [name]`, `/portfolio use NAME`, `/portfolio share CLIENT trader`, `/portfolio shared`.

### Fees and slippage

Every fill (market, pending, recurring DCA, margin open/close) uses a **slipped** price and a **taker fee** for that exchange:

| Venue | Fee | Slippage |
|-------|-----|----------|
| Binance / Bybit | 0.10% | 0.05% |
| Coinbase | 0.60% | 0.08% |

- **Buy fill:** `last × (1 + slip)`. Cash debit is `qty × fill × (1 + fee)`. Tax-lot unit cost is `fill × (1 + fee)`.
- **Sell fill:** `last × (1 − slip)`. Cash credit is `qty × fill × (1 − fee)`. Realized PnL uses net sell `fill × (1 − fee)` minus lot cost.
- **Pending buy reservation:** `remaining × slipped trigger × (1 + fee)` so the reserved cash covers the fee (and worst-case slip) when the order fills.
- Trades expose `price` (slipped fill), `fee` (quote), and `lastPrice` (pre-slip last).

`GET /api/v1/portfolio/trading-costs` lists the rates. MCP: `get_paper_trading_costs`.

### Tax lots (FIFO / LIFO)

Every **buy** (market, pending fill, recurring DCA) opens a lot: remaining quantity, **fee-inclusive** unit cost, and open time. A **sell** consumes lots in `lotMethod` order (`fifo` default, or `lifo`). Realized PnL is `(netSellPrice − lotPrice) × qty` for each consumed piece (net sell is after the sell fee). Partial sells leave leftover quantity on the same lot (original quantity is kept). Existing books get one synthetic lot per open position on migrate so old inventory still sells.

Pass `lotMethod` on `POST /orders` (market sells and sell pending/OCO/bracket exits). `GET /lots` lists remaining (or closed) lots. Trade history and cash/qty balances stay consistent with fee-inclusive fills.

MCP: `list_portfolio_lots`; `place_portfolio_order` / pending sell tools accept `lotMethod`. Telegram: `/sell SYMBOL QTY [ex] [fifo|lifo]`.

### Deposits and withdrawals

Users can add or remove virtual cash after create (`POST /deposits`, `POST /withdrawals`). Withdrawals use **available** cash only (not reserved for open orders). Each action is stored on `GET /cash-movements` (amount, kind, cash after, optional note, timestamp). Creating a portfolio writes an opening `deposit` row (`note=Opening balance`).

**Internal transfer:** `POST /transfers` moves available cash from one of **your** books to another. Owner only (shared traders/viewers cannot). Both ledgers get a row: `transfer_out` / `transfer_in` with `counterpartyPortfolioId`, `counterpartyPortfolioName`, and `peerMovementId` — not a deposit or withdrawal. Contributed capital moves with the cash so neither book's trading P&L changes.

**P&L:** `totalPnL = equity − startingBalance − netDeposits`. Depositing or withdrawing is not trading profit/loss. `contributedCapital` is starting + net deposits.

MCP: `deposit_portfolio_cash`, `withdraw_portfolio_cash`, `transfer_portfolio_cash`, `list_portfolio_cash_movements`. Telegram: `/deposit`, `/withdraw`, `/transfer`, `/cash`.

### Performance history

`GET /api/v1/portfolio/performance?period=1w` returns how total equity moved over the lookback, for a chart and for “since start of window” P&L.

| Field | Meaning |
|-------|---------|
| `period` | `1d` (24h), `1w` (7d, default), `1m` (30d), `3m` (90d) |
| `startEquity` / `endEquity` | Mark-to-market equity at window start (carry-forward last sample) and now |
| `changeAmount` | `endEquity - startEquity` (portfolio currency; equity jumps if you deposit/withdraw) |
| `changePct` | Percent vs `startEquity`; `null` if start is ~0 |
| `partial` | Portfolio created after the requested window start |
| `points[]` | `{ t, equity, cashBalance, positionsValue, marginEquity }` time series (includes live last point) |

**How it is stored:** a background `SnapshotWorker` writes one SQLite row per client per 15-minute UTC bucket (`PORTFOLIO_SNAPSHOT_INTERVAL`, default `15m`). Creating a portfolio records the starting equity immediately. Samples older than `PORTFOLIO_SNAPSHOT_RETENTION` (default 100 days) are pruned. History only exists while the server is running — gaps stay as missing buckets; the API still carry-forwards the last known equity.

MCP: `get_portfolio_performance`.

### Margin (isolated & cross)

See `docs/features/paper-margin.md`. Modes: `isolated` (default) vs `cross`; mode locked while open positions or pending margin orders exist. Isolated supports add/remove margin. Limit orders reserve margin until fill/cancel. Snapshot includes `marginMode`, `marginLocked`, `marginUnrealizedPnL`, `marginEquity`, `reservedMargin`, `marginPositions`.

### Recurring buys (DCA)

| Field | Description |
|-------|-------------|
| `name` | Optional label (`Salary Day Buy`); default `"SYMBOL frequency"` |
| `symbol` + `exchange` | Coin pair to buy |
| `amount` | Cash notional spent each run (`qty = amount / (slippedFill × (1 + fee))`) |
| `frequency` | `daily` \| `weekly` \| `monthly` \| `interval` |
| `weekday` | Weekly: `monday`…`sunday` (UTC) |
| `dayOfMonth` | Monthly: 1–31 (salary day; clamped on short months) |
| `intervalHours` | Interval: 1–168 hours (e.g. `12`) |
| `startAt` | Optional first run (RFC3339); default now |

**Lifecycle:** create (active) → pause / resume → delete. Failed runs (e.g. insufficient cash) keep the plan active and only that period is recorded as failed.

**Safety:**
- `UNIQUE(plan_id, period_key)` claim so restarts and concurrent workers cannot double-buy the same period.
- Missed windows run **only the latest** due slot (no backlog of intermediate buys).
- Worker interval: `RECURRING_BUY_INTERVAL` (default `30s`).

### Order types (`POST /api/v1/portfolio/orders`)

| `type` | Required fields | Fill condition (last price) | Side | Reservation |
|--------|-----------------|------------------------------|------|-------------|
| `market` (default) | `side`, `quantity` | Immediate at last price plus adverse slippage | buy / sell | Uses **available** cash/qty only (buy cash must cover fill + fee) |
| `limit_buy` | `quantity`, `triggerPrice` | last ≤ trigger | buy | Reserves `qty × slipped trigger × (1+fee)` cash |
| `limit_sell` | `quantity`, `triggerPrice` | last ≥ trigger | sell | Reserves `quantity` of position |
| `stop_loss` | `quantity`, `triggerPrice` | last ≤ trigger | sell | Reserves `quantity` of position |
| `trailing_stop` | `quantity`, `trailType`, `trailValue` | last ≤ ratcheted stop | sell | Reserves `quantity` of position |
| `oco` | `quantity`, `takeProfitPrice`, `stopLossPrice` | TP: last ≥ TP; SL: last ≤ SL | sell + sell | Reserves **once** for the pair |
| `bracket` | `quantity`, `triggerPrice` (entry), `takeProfitPrice`, `stopLossPrice` | Entry limit_buy; exits as OCO after fill | buy + sell pair | Entry reserves cash; exits reserve position only when active |

### Bracket orders (entry + TP/SL)

Place take-profit and stop-loss **with** a limit-buy entry. Exit legs start as `status=pending` (not marketable) until the entry fills.

| Rule | Behavior |
|------|----------|
| Place | Entry `open` limit_buy; TP + SL `pending` with size 0, linked by `bracketId` |
| Partial entry fill | Exit legs activate for **filled amount only**; size grows as more entry fills |
| Full entry fill | Exits open for full quantity |
| Exit fill | TP/SL are OCO: full fill cancels peer; partial reduces peer remaining — no double sell of the same qty |
| Cancel entry | Cancels pending/open exits (`bracket_entry_canceled`) |
| Prices | `takeProfitPrice` > `triggerPrice` (entry) > `stopLossPrice` |

### Trailing stop

Sell stop that follows price higher and never moves back down.

| Field | Meaning |
|-------|---------|
| `trailType` | `percent` (fraction below peak, e.g. `0.05` = 5%) or `offset` (fixed price units below peak) |
| `trailValue` | Distance in the chosen mode |
| `trailPeak` | High-water mark (seeded from last price at place; only increases) |
| `triggerPrice` | Current stop = `peak × (1 − percent)` or `peak − offset` |

**Rules**
- As last price makes a new high, peak and stop ratchet **up** only.
- Pullbacks that stay above the stop do **not** lower the stop.
- When last ≤ stop, the order **fills once** (market sell at last) and closes (status `filled`).
- If price **gaps** through the stop in one update (e.g. peak 120, stop 114, next last 100), it still triggers (`last ≤ stop`).

### OCO (one-cancels-the-other)

Linked **take-profit** (`limit_sell` at `takeProfitPrice`) and **stop-loss** (`stop_loss` at `stopLossPrice`) for the **same quantity**.

| Rule | Behavior |
|------|----------|
| Place | Both legs open with shared `ocoGroupId` / `ocoPeerId`; position size reserved **once** (not 2×) |
| Full fill | Filling leg → `filled`; peer → `canceled` with `cancelReason=oco_peer_filled` |
| Partial fill | Filling leg reduces remaining; peer remaining (and reserved size) **matches** the same remaining qty — one cash/position update only |
| Same tick | If both levels are crossed in one price update, **only stop-loss fills** (peer TP canceled if full); balance/asset never change twice |
| User cancel | Canceling one open leg cancels the peer (`oco_group_canceled`) |

Optional pending fields:
- `timeInForce`: `gtc` (default), `ioc`, or `fok` (single-leg pending; OCO is GTC)
- `expiresAt`: RFC3339 timestamp (**GTC only**); when reached the order is canceled and unused reservation is released (OCO expires as a group)

### Amend open limit / stop (`PATCH /api/v1/portfolio/orders/{id}`)

Change **price** and/or **remaining size** without canceling. Same order id; partial fills and trade history stay attached.

| Allowed | Not allowed (v1) |
|---------|------------------|
| `limit_buy`, `limit_sell`, `stop_loss` | `trailing_stop` (edit trail, not trigger) |
| `status=open`, `timeInForce=gtc` | IOC / FOK, filled / canceled / rejected |
| Standalone orders | OCO legs, bracket legs, margin limits |

**Body:** at least one of `triggerPrice`, `remainingQuantity` (remaining must stay `> 0` — use DELETE to cancel). Original `quantity` becomes `filledQuantity + remainingQuantity`. Reservations are recalculated atomically (`remaining × slipped trigger × (1+fee)` cash on buys; remaining qty on sells). Extra cash/qty is checked against available **plus this order’s current reservation**.

**Edit-screen GET** returns `lastPrice`, `editable`, and `amend` hints (`availableCashForOrder`, `availableQuantityForOrder`, `maxRemainingQuantity`, `minRemainingQuantity`) so the form can validate without guessing.

**Concurrency:** store compare-and-set on open + remaining + trigger. If the filler filled or canceled in between → **409**. After a successful amend, one fill attempt runs immediately if the new price is already marketable.

### Cancel all (`POST /api/v1/portfolio/orders/cancel-all`)

One action for the open-orders table (and AI tools): wipe every resting paper order, or only one pair.

| Scope | Body | Effect |
|-------|------|--------|
| All markets | `{}` or omit `symbol` | Every `open` + inactive bracket-exit `pending` order for the client |
| One market | `{ "symbol": "BTCUSDT", "exchange": "binance" }` | That pair only (`exchange` defaults to binance) |
| One venue | `{ "exchange": "binance" }` | Every pair on that exchange |

Releases unused cash/position reservations in one store transaction. `canceled: 0` is success (nothing open). Does **not** cancel margin limit orders. Cancel reason is `user`.

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
- Amend uses `status = open` plus remaining/trigger compare-and-set so a concurrent fill cannot be overwritten (HTTP **409**).
- Background filler runs on `PORTFOLIO_ORDER_CHECK_INTERVAL` and once on process start.
- Recurring buy worker runs on `RECURRING_BUY_INTERVAL` and once on process start.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go`, `recurring_buy.go`, `allocation.go`, `risk.go` |
| Store | `backend/internal/adapter/portfoliostore` |
| Service | `backend/internal/service/portfolio` |
| Filler | `backend/internal/service/portfolio/filler.go` |
| Recurring worker | `backend/internal/service/portfolio/recurring_worker.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio.go`, `portfolio_recurring.go`, `portfolio_allocation.go`, `portfolio_margin.go` |
| MCP | portfolio tools + `get_portfolio_order`, `amend_portfolio_order`, `cancel_all_portfolio_orders` + recurring buy tools + allocation basket / rebalance tools + `place_margin_order`, `list_margin_positions`, `close_margin_position`, `set_margin_brackets`, `list_margin_orders`, `cancel_margin_order`, `list_margin_trades` |

## Config

| Env | Default |
|-----|---------|
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` |
| `RECURRING_BUY_INTERVAL` | `30s` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ -run 'Portfolio|Recurring|Allocation' -count=1
```