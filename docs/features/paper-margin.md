# Feature: Paper margin trading (isolated & cross)

## Problem / goal

Users want leveraged long/short paper trading without real money: market and limit opens, leverage 1x–10x, **isolated or cross** margin mode, correct margin / available balance / PnL / liquidation, auto liquidation, partial closes, add/remove margin (isolated), and stop-loss / take-profit.

## Behavior

### Margin modes (account-wide)

| Mode | Meaning |
|------|---------|
| `isolated` (default) | Only margin assigned to a position backs that position. Add/remove margin allowed. |
| `cross` | Wallet equity is shared; free margin includes open unrealized PnL; liq uses shared equity. |

`PUT /api/v1/portfolio/margin/mode` changes mode. **Blocked** while any open margin position **or** pending margin limit order exists.

### Open
- **Side:** `long` or `short`; **leverage:** 1–10; **type:** `market` or `limit`
- **Initial margin:** `qty * price / leverage` debited from available cash
- **Limit:** reserves required margin until fill, cancel, or reject (released on cancel)

### Liquidation (maintenance 0.5% of notional)
- **Isolated:** `liq` from assigned margin: long `entry − (margin − maint)/qty`
- **Cross:** per-position liq from shared equity vs total maintenance; if equity &lt; total maint, liquidate open positions
- Worker auto-closes when mark crosses liquidation (reason `liquidation`)

### Add / remove margin (isolated only)
- `POST .../positions/{id}/margin` with `delta` (+ add from cash, − return to cash)
- Cannot go below initial margin for remaining size
- Liquidation price recalculated after adjust

### Borrowing & interest
- **Long:** borrowed **cash** = notional − margin (`debtAsset=quote`); principal and interest shown separately
- **Short:** borrowed **base coins** = quantity (`debtAsset=base`)
- **Interest:** simple hourly on principal (`DefaultMarginHourlyInterestRate`); applied on worker tick and when reading/closing/repaying
- **Liquidation** uses principal + interest (long liq rises as interest grows)
- **Borrow limit:** total debt notional ≤ `startingBalance * 9` (10x − own capital)

### Partial close
- Releases **proportional** margin and **proportional debt** (principal + interest); realizes partial PnL; liq recalculated

### Repay without close
- `POST .../repay` with `amount` in debt units (quote cash for long, coins for short)
- Pays **interest first**, then principal
- Short repay buys coins at mark from available cash

### Stop-loss / take-profit
- Optional on open or `PUT .../brackets`; worker closes at market when hit

### Equity
`equity = cashBalance + spotPositionsValue + marginLocked + marginUnrealizedPnL`

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

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/margin.go` |
| Store | `backend/internal/adapter/portfoliostore/margin.go` |
| Service | `backend/internal/service/portfolio/margin.go` |
| Worker | `ProcessMarginMaintenance` via portfolio filler |
| HTTP | `backend/internal/transport/http/handler/portfolio_margin.go` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ -run Margin -count=1
```

## Limitations

- No funding rates or trading fees
- Cross liquidation of under-maint account closes open positions (paper simplification)
- Informational simulation — not real money / not financial advice
