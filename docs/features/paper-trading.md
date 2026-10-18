# Paper trading / virtual portfolio

## Goal

Simulated portfolios with starting cash, market buy/sell at last price, **pending limit/stop orders with reservations and partial fills**, open positions, realized/unrealized P&L, and trade history. **Not real money.** Data is stored in SQLite and survives restarts.

### Product web UI

The React app exposes paper trading at **`/portfolio`** (`PortfolioPage`): multi-book select/create, cash deposit/withdraw, market orders, positions, open pending orders (cancel), trade history, cash ledger, and a performance chart. Cash you add or remove is not part of that line. Coin detail (`/markets/:exchange/:symbol`) includes a **Paper trade** panel that places market orders against the selected book. On a crypto market ticket, this venue, best cost, and a time split sit side by side with the fee and the slippage for the same size. Choosing a column is what gets placed. When that book holds the coin, the chart marks the fill and the nearer estimated liquidation pocket, and the Trade tab states the percent between them. A margin position also marks its paper liquidation price. A missing venue leaves the pocket unmarked. A crypto market order stores the coin situation when it fills. A buy can name which clauses are the reason; only those are stored. Omit that list and the fill keeps the whole read. A sell keeps the whole read. A resting order keeps the placement read on the order and stores a new read on the fill. The fill of a buy that named clauses keeps those same clauses. Trade history shows that fill read and says it is the market at the fill. An open crypto position lists every remaining buy with its size, fill time, and stored read, and names which of those clauses the live read has left. Entered as and the situation alert stay on the newest buy. On `/alerts`, that alert names its last fire, and says when the newest buy stored no read or no open buy is left. The live situation sits under that list. The live Now line also says how long a clause has held when the archive reaches a different kind. The stored entry does not. A sell names the same difference against each buy it closed. Above the tabs, the desk groups closed crypto lots by the kind stored on the buy: how many closes, how many were profitable, and the sum of realized profit. The same close can appear on more than one row. A crypto buy with no stored read is counted as skipped. Under that record, each entry kind splits into the closes that sold while the sell read still matched and the closes that sold after it left. A sell with no stored read stays out of both groups. Nasdaq and BIST orders skip the read and stay out of both records. See [`coin-situation.md`](coin-situation.md).

Trade history and each open position also show the venue price change **1h, 4h, and 24h after the fill**. A trade row uses the slipped fill price and the fill time. An open position uses the newest lot that still has size, including Nasdaq and BIST, not the average cost. The sample is the close of the one-minute bar that contains that instant, and only after that minute has closed. Until then the window stays **pending**. A finished window with no minute close stays **blank** and is not zero. A real unchanged price is **0%**. The percent is the venue move, not profit: a sell can show a positive percent when price rose after the sale. Realized P&L stays in its own column. Public equity charts only keep about a day of one-minute bars, so an older stock fill stays blank rather than substituting an hourly close. The portfolio service caches each minute close for about 45 seconds. A failed candle read leaves that window blank and still returns the book. With **more than one book**, snapshot queries skip and trade/cash/margin submits stay disabled until a book is selected (the API returns 400 without `portfolioId`). Realtime portfolio WS updates are subscribed when a book is selected.

Spot, pending, margin, and recurring-buy symbols must use the **same quote asset as the portfolio currency** (default `USDT`). `ETHBTC` on a USDT book is rejected. When a live mark cannot be fetched, margin positions are **not** marked at entry (liquidation is skipped until a real last price exists).

Live order/position/cash updates for a selected book (and price ticks for selected coins) go over **`GET /api/v1/ws`** — see [`realtime.md`](realtime.md).

Backup and restore owned books with the user data export/import jobs (`sections=portfolios`, merge or replace) — see [`user-data-export.md`](user-data-export.md) and [`user-data-import.md`](user-data-import.md).

### Idempotency keys

`POST /orders` (market, pending, OCO, bracket), `POST /margin/orders`, `POST /margin/positions/{id}/close`, and **`POST /transfers`** accept `Idempotency-Key` (or `X-Idempotency-Key`) header or `idempotencyKey` in the JSON body. A retry with the **same key and same request** returns the original trade/order/transfer instead of creating a second one (`replayed=true` on transfers). The same key with a **different** request returns **409**. Keys are scoped to the paper book, last **24 hours**, and may use letters, digits, `-`, `_`, `.`, `:` (max 128). MCP tools take `idempotencyKey` the same way.

Transfer tries are stored so you can see which request **completed** and which **came again**: `GET /api/v1/portfolio/transfers/attempts` (`completed` / `replayed` / `conflict`). The web cash panel sends a new key per click and reuses it if the connection drops before a success response.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio` | Create a named book (`startingBalance`, optional `name`, `currency`) |
| `GET` | `/api/v1/portfolios` | List books for the client |
| `PATCH` | `/api/v1/portfolios/{id}` | Rename a book |
| `DELETE` | `/api/v1/portfolios/{id}` | Delete a book and all of its data |
| `GET` | `/api/v1/portfolio` | Snapshot of the selected book (`portfolioId` / `X-Portfolio-Id`) |
| `GET` | `/api/v1/portfolio/fill-pocket` | Open fill against the nearer estimated liquidation pocket (`exchange`, `symbol`). `gapPct` is the percent from the fill to the pocket. `marginLiqPrice` is the paper liquidation price. A missing venue leaves the pocket blank |
| `POST` | `/api/v1/portfolio/deposits` | Add virtual cash (`amount`, optional `note`) |
| `POST` | `/api/v1/portfolio/withdrawals` | Withdraw available cash (`amount`, optional `note`) |
| `POST` | `/api/v1/portfolio/transfers` | Move cash between your own books (`fromPortfolioId`, `toPortfolioId`, `amount`, optional `idempotencyKey`) — owner only |
| `GET` | `/api/v1/portfolio/transfers/attempts` | Transfer request tries (completed / replayed / conflict) |
| `GET` | `/api/v1/portfolio/cash-movements` | Deposit / withdraw / transfer history (newest first; includes opening) |
| `GET` | `/api/v1/portfolio/performance` | Equity series + period P&L (`period=1d\|1w\|1m\|3m`) |
| `GET`/`PUT`/`DELETE` | `/api/v1/portfolio/risk-limits` | Optional daily-loss %, max coin weight %, and max total exposure % (block new buys/margin only) |
| `POST` | `/api/v1/portfolio/orders` | Market or pending order (see below); optional `Idempotency-Key`. `spread=true` with `exchange=all` starts timed slices (`spreadMinutes`, `sliceCount`) and returns 201 |
| `GET` | `/api/v1/portfolio/spread-orders` | List spread orders (`status` default `open`) |
| `DELETE` | `/api/v1/portfolio/spread-orders/{id}` | Cancel an open spread order and release the unused lock |
| `GET` | `/api/v1/portfolio/orders` | List pending orders (`status` default `open`) |
| `GET` | `/api/v1/portfolio/orders/{id}` | One order + last price + amend hints for the edit screen |
| `PATCH` | `/api/v1/portfolio/orders/{id}` | Amend open GTC limit/stop in place (`triggerPrice`, `remainingQuantity`) |
| `POST` | `/api/v1/portfolio/orders/cancel-all` | Cancel all open orders, or one market (`symbol` + optional `exchange`) |
| `DELETE` | `/api/v1/portfolio/orders/{id}` | Cancel open pending order (releases unused reservation) |
| `GET` | `/api/v1/portfolio/trades` | Trade history (`limit`, `offset`); pending fills include `pendingOrderId`; `price` is slipped fill, `fee` is quote taker fee, `lastPrice` is pre-slip last; `moves` is the 1h/4h/24h venue price change after the fill (`pending`, `ready`, or `missing`) |
| `GET` | `/api/v1/portfolio/situation-record` | Closed crypto lots grouped by the situation stored on the buy. Each row is one entry kind and lists the closes that stored it, newest first. The same close can appear on more than one row. A crypto buy with no stored read is skipped. Equities and open positions are left out |
| `GET` | `/api/v1/portfolio/situation-leave` | Each entry kind split into closes whose sell read still matched and closes that sold after it left. A blank sell read is `unreadable` and stays out of both groups. The same close can appear on more than one row. Equities and open positions are left out |
| `GET` | `/api/v1/portfolio/trading-costs` | Per-exchange paper taker fee and slippage (`?exchange=` optional) |
| `POST` | `/api/v1/portfolio/orders/execution-compare` | Fee and slippage for one crypto size on one venue, as best cost, and as a time split. Nothing is written |
| `GET` | `/api/v1/portfolio/lots` | Tax lots (`status=open\|closed\|all`, optional `exchange`/`symbol`) |
| `POST` | `/api/v1/portfolio/recurring-buys` | Create named recurring buy (DCA) plan |
| `GET` | `/api/v1/portfolio/recurring-buys` | List plans |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}` | Get plan |
| `PATCH` | `/api/v1/portfolio/recurring-buys/{id}` | Update name / amount / schedule / timeZone / maxPrice / budget / endDate |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/pause` | Pause plan |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/resume` | Resume plan |
| `DELETE` | `/api/v1/portfolio/recurring-buys/{id}` | Delete plan (+ run history) |
| `GET` | `/api/v1/portfolio/recurring-buys/{id}/runs` | Execution history (missed slots + on_time/late) |
| `POST` | `/api/v1/portfolio/recurring-buys/{id}/runs/{runId}/complete` | Buy a missed slot now |
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

Tenancy uses the same `clientId` / `X-Client-Id` model as watchlists. Each client may own **up to 20 named paper books**. The first book keeps id = `clientId` and default name `Main` (legacy). Additional books get a UUID id. All action routes (`/orders`, `/deposits`, `/performance`, …) accept optional `portfolioId` query, `X-Portfolio-Id` header, or body field. Omit it when there is exactly one book; if several exist, the API returns 400 until you select one. The recurring-buy worker always places on the book that owns the plan (it does not omit `portfolioId`).

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

### Fees and the book

A crypto market fill, and a resting order when it triggers, walks that venue's visible spot book. The fill price is the average of the levels taken. A buy walks asks from low to high. A sell walks bids from high to low. A limit only takes prices at or better than the limit. A size the visible book cannot hold is not invented past the last level: a market order is rejected, and a resting order stays open. Equities have no book, so they still use last price plus the fixed slippage below. A liquidation, stop, or take-profit that the book cannot hold also falls back to that slippage so the protective close still happens.

Every fill still pays a **taker fee**. The fixed slippage rates are:

| Venue | Fee | Slippage |
|-------|-----|----------|
| Binance / Bybit | 0.10% | 0.05% |
| Coinbase | 0.60% | 0.08% |

- **Crypto buy:** average of the visible asks. Cash debit is `qty × fill × (1 + fee)`. Tax-lot unit cost is `fill × (1 + fee)`.
- **Crypto sell:** average of the visible bids. Cash credit is `qty × fill × (1 − fee)`. Realized PnL uses net sell `fill × (1 − fee)` minus lot cost.
- **Best cost:** `exchange=all` on a spot market order (preview and place) splits the quantity across Binance, Coinbase, and Bybit. A buy takes the lowest `price × (1 + fee)`. A sell takes the highest `price × (1 − fee)`. Levels on one venue fold into one trade. Cash moves once. A size the books cannot hold fills nothing. A sell only uses coins already held on that venue. A missing book is named in `missingVenues` and its size is not zero. `compareExchange` prices the whole quantity on one venue; `savedCash` is set when both can fill. A USDT book books a Coinbase USD fill 1:1 and stores the position as `BTC-USD` on Coinbase. `maxSlippagePct` of `0.5` keeps a buy at or below 0.5% above the best ask and a sell at or above 0.5% below the best bid. Prices past that cap are not taken. When the size inside the cap is short, nothing is filled, and the preview says so. Margin, equities, and resting orders stay on one venue. The ticket shows **Best cost** on a crypto market order.
- **Spread:** `spread=true` with best cost does not fill the size in one pass. `spreadMinutes` is the clock from the first slice to the last (default 10, from 1 to 1440). `sliceCount` is how many times the books are read (default 10, from 2 to 60). The gap is that duration divided by one less than the slice count, so 15 slices over 30 minutes are about 2 minutes 9 seconds apart, and 5 slices over 5 minutes are 75 seconds apart. A gap shorter than 15 seconds is rejected. The first slice is due when you place it and runs on the next worker pass. The last slice is due when the minutes end. Each slice reads Binance, Coinbase, and Bybit again and takes the cheapest fee-adjusted size still open (`remaining / slices left`). A thin book can start when one slice fits, even if the whole size does not. A buy locks cash up front from the best ask (the slippage cap, or 1% when there is no cap) including the Coinbase taker fee, and releases the unused share as slices fill. A sell locks quantity on the venues that already hold the coin, in proportion to what is free there, and each slice can only sell that lock. A slice the book cannot fill counts as an attempt and leaves the quantity for later slices. After the last attempt, leftover size is canceled and the lock is released. Cancel does the same for what has not filled. Fills already taken stay. A restart fills at most one overdue slice per pass, then waits one gap, so it does not dump the rest at once. The preview shows the schedule, the cash or quantity lock, and the first slice on the current book. `routeLegs` are that first slice. `compare` is that slice against one venue. Nothing is written until you place it. The ticket sets the minutes and the slices. The open orders tab lists working spread orders. `GET /api/v1/portfolio/spread-orders`, `DELETE /api/v1/portfolio/spread-orders/{id}`. MCP `list_spread_orders`, `cancel_spread_order`. `preview_portfolio_order` and `place_portfolio_order` take `spread`, `spreadMinutes`, and `sliceCount`.
- **Equity fill:** `last × (1 ± slip)` with the rates below.
- **Pending buy reservation:** `remaining × slipped trigger × (1 + fee)` so the reserved cash covers the fee. When the order triggers, the fill walks the book and only takes prices at or better than the limit.
- Trades expose `price` (book average, or slipped equity fill), `fee` (quote), and `lastPrice` (the last print).

`GET /api/v1/portfolio/trading-costs` lists the rates. MCP: `get_paper_trading_costs`.

### Tax lots (FIFO / LIFO)

Every **buy** (market, pending fill, recurring DCA) opens a lot: remaining quantity, **fee-inclusive** unit cost, and open time. A **sell** consumes lots in `lotMethod` order (`fifo` default, or `lifo`). Realized PnL is `(netSellPrice − lotPrice) × qty` for each consumed piece (net sell is after the sell fee). Partial sells leave leftover quantity on the same lot (original quantity is kept). Existing books get one synthetic lot per open position on migrate so old inventory still sells.

Pass `lotMethod` on `POST /orders` (market sells and sell pending/OCO/bracket exits). `GET /lots` lists remaining (or closed) lots. Trade history and cash/qty balances stay consistent with fee-inclusive fills.

MCP: `list_portfolio_lots`; `place_portfolio_order` / pending sell tools accept `lotMethod`. Telegram: `/sell SYMBOL QTY [ex] [fifo|lifo]`.

### Deposits and withdrawals

Users can add or remove virtual cash after create (`POST /deposits`, `POST /withdrawals`). Withdrawals use **available** cash only (not reserved for open orders). Each action is stored on `GET /cash-movements` (amount, kind, cash after, optional note, timestamp). Creating a portfolio writes an opening `deposit` row (`note=Opening balance`).

**Internal transfer:** `POST /transfers` moves available cash from one of **your** books to another. Owner only (shared traders/viewers cannot). Books must share a settlement currency (USDT/USDC/USD are treated as the same). Cross-margin books cannot transfer cash that would leave equity under maintenance (same brake as withdraw). Both ledgers get a row: `transfer_out` / `transfer_in` with `counterpartyPortfolioId`, `counterpartyPortfolioName`, and `peerMovementId` — not a deposit or withdrawal. Contributed capital moves with the cash so neither book's trading P&L changes.

**P&L:** `totalPnL = equity − startingBalance − netDeposits`. Depositing or withdrawing is not trading profit/loss. `contributedCapital` is starting + net deposits.

MCP: `deposit_portfolio_cash`, `withdraw_portfolio_cash`, `transfer_portfolio_cash`, `list_portfolio_cash_movements`. Telegram: `/deposit`, `/withdraw`, `/transfer`, `/cash`.

### Performance history

`GET /api/v1/portfolio/performance?period=1w` returns account equity over the lookback, plus trading P&L since the start of that window. Deposits, withdrawals, and transfers between your books are external cash. They move equity and they are not profit or loss. The opening balance is the starting capital, not a later deposit.

| Field | Meaning |
|-------|---------|
| `period` | `1d` (24h), `1w` (7d, default), `1m` (30d), `3m` (90d) |
| `startEquity` / `endEquity` | Mark-to-market account equity at the baseline and now. A deposit still raises `endEquity` |
| `netFlow` | External cash added during the window (deposits + transfers in − withdrawals − transfers out) |
| `changeAmount` | Trading P&L: `(endEquity − startEquity) − netFlow` |
| `changePct` | Modified Dietz percent of that trading P&L. Cash added or removed counts in the base only for the share of the window it was in the book. `null` when that base is ~0 or negative |
| `maxDrawdown` | Largest peak-to-trough drop of the performance line (account equity with external cash removed). A withdrawal is not a drawdown. 0 when that line never fell |
| `maxDrawdownPct` | That drop as a percent of the performance peak; `null` when the peak is ~0 |
| `drawdownFrom` / `drawdownTo` | When the performance peak was printed and when the line reached the trough |
| `partial` | Portfolio created after the requested window start |
| `points[]` | `{ t, equity, cashBalance, positionsValue, marginEquity, externalFlow }`. `equity` is account value. The performance level is `equity − externalFlow` |

The portfolio chart plots `equity − externalFlow` and shows `changeAmount`. Account equity on the summary strip still includes cash you added.

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
| `market` (default) | `side`, `quantity` | Immediate. Crypto walks the visible spot book (average price). Equities use last price plus adverse slippage. A size the book cannot hold is not filled | buy / sell | Uses **available** cash/qty only (buy cash must cover fill + fee) |
| `limit_buy` | `quantity`, `triggerPrice` | last ≤ trigger | buy | Reserves `qty × slipped trigger × (1+fee)` cash |
| `limit_sell` | `quantity`, `triggerPrice` | last ≥ trigger | sell | Reserves `quantity` of position |
| `stop_loss` | `quantity`, `triggerPrice` | last ≤ trigger | sell | Reserves `quantity` of position |
| `trailing_stop` | `quantity`, `trailType`, `trailValue` | last ≤ ratcheted stop | sell | Reserves `quantity` of position |
| `oco` | `quantity`, `stopLossPrice`, and `takeProfitPrice` or `takeProfitTargets` | Each target: last ≥ its limit; stop: last ≤ stop | sell | Reserves **once** for the whole size |
| `bracket` | `quantity`, `triggerPrice` (entry), `stopLossPrice`, and `takeProfitPrice` or `takeProfitTargets` | Entry limit_buy; exits after the fill | buy + sells | Entry reserves cash; exits reserve position only when active |

### Order preview (`POST /api/v1/portfolio/orders/preview`)

Prices the same spot body, or a margin open (`product=margin`, `side=long|short`, `type=market|limit`, `leverage` 1–10, `limitPrice` on a limit). The response is the slipped fill, taker fee, cash in or out, cash or quantity a resting order would lock, estimated sell profit from the open lots, and whether a risk limit or the borrow cap would block it.

Nothing is written: no trade, no reservation, no margin position. A draft that cannot be priced (bad quantity, missing trigger, bad bracket prices) is **400**. Not enough cash, a risk block, or the borrow cap is **200** with a flag. A sell is not blocked. The daily-loss figure is trading P&L since the UTC day baseline. Deposits, withdrawals, and transfers are not part of it. MCP `preview_portfolio_order`. A read-only assistant session can call that tool. It still cannot place the order.

A spot buy, limit buy, or bracket that names `reason`, and whose live read is `ready`, also returns `reasonHistory`. Each row is one marked clause at the kind that read says now. `still` is this book's past closes whose sell kept that kind. `left` is the closes whose sell changed it or left the clause blank. A change in print dollars with the same side stays in `still`. Both counts at zero means no past close stored that live kind. The same close can appear on more than one row. An unset reason, an empty reason, a sell, a margin order, an equity, and a blank, failed, or skipped read omit the list. If the close list cannot be read, the price is still returned and `reasonHistory` is omitted.

**Size from a risk percent.** Send `riskPercent` (greater than 0 and at most 100) and `stopLossPrice`. `quantity` is ignored. The size is the base amount that loses that percent of **portfolio equity** if the stop fills. The loss of one unit includes the taker fee and adverse slippage on the entry and on the stop, the same loss a paper fill would book. Leverage does not change that loss. It changes the margin and the liquidation price.

| Order | Entry used | Stop is placed |
|-------|------------|----------------|
| Spot market buy | Last price | No. The stop only sizes the buy |
| Spot `limit_buy` | Trigger | No |
| Spot `bracket` | Entry trigger | Yes, with the bracket |
| Margin long or short, market | Last price | Yes, on the open |
| Margin long or short, limit | Limit price | Yes, on the open |

A spot sell, stop, trailing stop, or OCO is **400**. A missing stop is **400**. If the sized quantity is below `1e-8` or above `1e9`, the preview is **400**. The response `quantity` is the size, and `sizing` repeats the budget, the loss at the stop, `stopAttached`, and `stopBeyondLiquidation`. The last flag is true when a margin stop is at or past the liquidation price shown on the same preview, so the stop would not be the exit. Not enough cash for that size is still **200** with `insufficientCash`.

### Execution compare (`POST /api/v1/portfolio/orders/execution-compare`)

Prices one crypto spot size three ways. Nothing is written.

| Row | What it walks |
|-----|----------------|
| `market` | The venue you name (`binance`, `coinbase`, or `bybit`), the whole size, now. No slippage cap |
| `best_cost` | Binance, Coinbase, and Bybit now. A buy takes the lowest `price × (1 + fee)`. A sell takes the highest `price × (1 − fee)` |
| `spread` | That same best-cost walk, once per slice, as if each slice finds this book again |

`spreadMinutes` and `sliceCount` default to 10 slices over 10 minutes. The gap is the minutes divided by one less than the slice count. A gap under 15 seconds, a count outside 2–60, or a slice below the smallest fill sets that row to `schedule` and still returns the other two prices.

`touchPrice` is the best ask or bid across the books that loaded. `slippagePct` is how far the average fill sits from that price, in percent, before the fee. `0.5` means 0.5% worse. `fee` is the taker fee in the quote currency. `feePct` is that fee as a percent of notional. Cash includes the fee.

`savedVsMarket` is set only when that row and the market row can both fill. Positive means a buy spends less, or a sell receives more, than the single venue.

`maxSlippagePct` of `0.5` caps best cost and the time split. The market row ignores it. A path the book cannot hold has a status other than `filled` and no cash. A missing book is named in `missing` and is not treated as zero. A sell only uses coins already held. The time split locks those coins the way a placed spread order does, then assumes the book refills between slices. The coins do not refill. If the book does not refill, the time split costs the same as best cost. Later prices are not known.

Equities are rejected. Margin stays on one venue. MCP `compare_order_execution`. A read-only assistant session can call it. The crypto market ticket shows the three rows and places the one you choose. A filled row names each venue with the base size it takes, such as `binance 0.4, bybit 0.6`. The rows show the clock of the last successful read. While a newer read is in flight, including the automatic refresh, those rows stay and the ticket says it is updating.

The response `id` is kept for two minutes. Place sends it back as `executionQuoteId`. When the size, side, venue, slippage cap, and slice schedule still match, that quote is kept with the order. Trade history then shows the quoted fee and the quoted slippage in quote currency beside what the fills paid. The difference is actual minus expected. Positive means the fill cost more than the quote. Slippage uses the touch from that comparison, before the fee. A market order and a best-cost order show it on the newest fill as soon as they finish. A spread order shows it when the order is filled or canceled, including a cancel that kept earlier slices. A smaller filled size than the quoted size is named. A missing or different id still places the order and leaves the line off. One quote is kept for one order. MCP `list_portfolio_trades` returns the same `costCheck`.

### Bracket orders (entry + TP/SL)

Place take-profit and stop-loss **with** a limit-buy entry. Exit legs start as `status=pending` (not marketable) until the entry fills.

| Rule | Behavior |
|------|----------|
| Place | Entry `open` limit_buy; each take-profit and the stop `pending` with size 0, linked by `bracketId` |
| Partial entry fill | Each target opens for its percent of the **filled** size. The stop opens for that whole fill |
| Later entry fill | Targets grow by their percent of the new coins. A target that already filled its earlier share reopens for the rest |
| Target fill | The stop quantity becomes the coins still held. Other targets stay |
| Move stop to the buy price | Optional `moveStopToEntry`. After the nearest target fills completely, the stop rises to the average entry fill. It never moves down and stays below the next open target |
| Stop fill | A full stop cancels the remaining targets. A partial stop shrinks them so they cannot sell more than what is left |
| Cancel entry | Cancels pending/open exits (`bracket_entry_canceled`) |
| Prices | Each target is above the entry, and the entry is above `stopLossPrice` |

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

One or more **take-profit** limit sells and one **stop-loss** for the same coins. `takeProfitPrice` is one target for the whole size. `takeProfitTargets` is a scale-out:

```json
"takeProfitTargets": [
  { "percent": 25, "offsetPercent": 5 },
  { "percent": 25, "offsetPercent": 10 },
  { "percent": 50, "offsetPercent": 15 }
]
```

`percent` is the share of the quantity. The shares sum to 100. `offsetPercent` of 5 is a limit 5% above the average cost (or the last price when that cost is missing). `price` is an absolute limit instead of a gain. A bracket uses the same list, and the gain is above the entry limit.

| Rule | Behavior |
|------|----------|
| Place | Every target and the stop are open with one `ocoGroupId`. The position is reserved **once** (the stop size, not the sum of the legs) |
| Target fill | That slice sells. The stop quantity becomes the coins still held. The other targets stay |
| Move stop to the buy price | Optional `moveStopToEntry`. After the nearest target fills completely, the stop trigger rises to the buy price. An OCO uses the average cost at placement. A bracket uses the average entry fill. The preview shows the entry limit for a bracket. The stop never moves down, and it stays below the next open target. A later entry fill can change the stored average without moving the stop again. A partial fill of the nearest target does not move it. A later target does not move it when a nearer target was canceled. If that nearest target is the whole size, the stop is canceled with the fill and the option does nothing. The buy price must sit strictly between the stop and the first target, or place and preview are rejected |
| Full stop fill | Stop → `filled`. Every open target → `canceled` with `cancelReason=oco_peer_filled` |
| Partial stop fill | The open targets shrink in proportion. The furthest target absorbs the rounding, so they cannot sell more than what is left |
| Same tick | If the stop and a target are both touched, **only the stop fills**. Several targets the price has already reached fill together, nearest first |
| User cancel | Canceling the stop, or the last target, cancels the rest of the group. Canceling one target while others remain leaves the stop on the coins still held |

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
- Fill path **re-reads** the order under the book lock and compare-and-sets `remaining_quantity` against the remaining the fill was **sized from**. A concurrent amend that changed remaining is `ErrConflict`; the next filler tick retries with a fresh snapshot so leftover size cannot sit open with `reserved_* = 0`.

### Accounting
- **Buy fill:** debit cash at fill price; increase position; average cost updated.
- **Sell fill:** credit cash; reduce position; realize `(price - avgCost) * qty`.
- **Equity:** cash + market value of positions (reserved cash is still part of cash until filled).

### Durability & safety
- Portfolios, positions, trades, pending orders, remaining size, reservations, and recurring buy plans/runs are in SQLite (`PORTFOLIO_DB_PATH`).
- Fill/cancel/reject use `status = open` predicates so a canceled order never fills and a completed order is not double-filled.
- Amend uses `status = open` plus remaining/trigger compare-and-set so a concurrent fill cannot be overwritten (HTTP **409**). The web amend modal sends `remainingQuantity` only when the user changed that field (a price-only edit does not restore a pre-fill size).
- Service-layer cash/position updates are **serialized per `clientId`** so HTTP orders, recurring buys, and the multi-symbol filler cannot last-write-wins the same wallet.
- Background filler runs on `PORTFOLIO_ORDER_CHECK_INTERVAL` and once on process start.
- Recurring buy worker runs on `RECURRING_BUY_INTERVAL` and once on process start.

## Web UI (`frontend/`)

| Surface | Notes |
|---------|--------|
| `/portfolio` spot ticket | Market, limit, stop, trailing stop, OCO, bracket; TIF GTC/IOC/FOK; lot FIFO/LIFO on sells. OCO and bracket targets are size % and gain % (for example 25% at +5%, 25% at +10%, 50% at +15%). The preview lists each target and the stop. The ticket shows fee, cash, and risk before the order is placed. A buy or a bracket can take a risk % and a stop; the preview fills the quantity |
| Open orders table | Cancel + **amend** (standalone GTC limit/stop: trigger and remaining qty) |
| `/portfolio` margin ticket | Isolated/cross mode, long/short 1–10x, market or limit, optional SL/TP. Same preview: margin, liquidation, borrow cap. A risk % and a stop size the open |
| Margin positions tab | Mark, liq, debt, uPnL, stop, take profit, full or partial market close. Edit or remove the stop and take profit on an open position |
| Coin detail compact ticket | Market + limit + stop (no OCO/bracket), with the same preview. A buy can be sized from a risk % and a stop |

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go`, `order_preview.go`, `risk_size.go`, `recurring_buy.go`, `allocation.go`, `risk.go` |
| Store | `backend/internal/adapter/portfoliostore` |
| Service | `backend/internal/service/portfolio` |
| Filler | `backend/internal/service/portfolio/filler.go` |
| Recurring worker | `backend/internal/service/portfolio/recurring_worker.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio.go`, `portfolio_preview.go`, `portfolio_recurring.go`, `portfolio_allocation.go`, `portfolio_margin.go` |
| Web | `frontend/src/components/organisms/PaperTradeForm`, `PaperMarginForm`, `molecules/OrderPreview`, `PortfolioOrdersTable`, `PortfolioMarginPositionsTable`, `pages/PortfolioPage`, `pages/CoinDetailPage` |
| MCP | `preview_portfolio_order` + portfolio tools + `get_portfolio_order`, `amend_portfolio_order`, `cancel_all_portfolio_orders` + recurring buy tools + allocation basket / rebalance tools + `place_margin_order`, `list_margin_positions`, `close_margin_position`, `set_margin_brackets`, `list_margin_orders`, `cancel_margin_order`, `list_margin_trades` |

## Config

| Env | Default |
|-----|---------|
| `PORTFOLIO_DB_PATH` | `data/portfolio.db` |
| `PORTFOLIO_ORDER_CHECK_INTERVAL` | `15s` |
| `RECURRING_BUY_INTERVAL` | `30s` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ -run 'Portfolio|Recurring|Allocation|Preview|QuoteFill' -count=1
```