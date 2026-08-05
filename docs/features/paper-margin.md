# Feature: Paper margin trading (isolated)

## Problem / goal

Users want leveraged long/short paper trading (similar to exchange margin) without real money: market and limit opens, leverage 1x–10x, correct margin / available balance / PnL / liquidation, automatic liquidation, partial closes, and stop-loss / take-profit.

## Behavior

### Open
- **Side:** `long` or `short`
- **Leverage:** integer 1–10
- **Type:** `market` (fill at last price) or `limit` (rest until trigger)
- **Initial margin:** `qty * price / leverage` (isolated)
- Debited from **available cash** (spot + margin reservations already excluded)
- Limit opens **reserve** margin until fill/cancel/reject (cash not debited until fill)

### Liquidation (isolated, maintenance 0.5% of notional)
- Long: `entry * (1 - 1/lev + mmr)`
- Short: `entry * (1 + 1/lev - mmr)`
- Worker auto-closes full size when last price crosses liquidation (reason `liquidation`)

### PnL
- Long unrealized: `(mark - entry) * qty`
- Short unrealized: `(entry - mark) * qty`
- On close: realize same formula at exit; release proportional margin to cash

### Partial close
- `POST .../close` with `quantity` &lt; open size releases proportional margin and realizes partial PnL; position stays open

### Stop-loss / take-profit
- Optional on open or via `PUT .../brackets`
- Worker full-closes at market when triggered
- Long: SL below / TP above entry; short inverted

### Equity
`equity = cashBalance + spotPositionsValue + marginLocked + marginUnrealizedPnL`  
(`cashBalance` already excludes locked margin)

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/portfolio/margin/orders` | Market open or limit rest |
| `GET` | `/api/v1/portfolio/margin/orders` | List orders |
| `DELETE` | `/api/v1/portfolio/margin/orders/{id}` | Cancel limit |
| `GET` | `/api/v1/portfolio/margin/positions` | Open positions |
| `GET` | `/api/v1/portfolio/margin/positions/{id}` | One position |
| `POST` | `/api/v1/portfolio/margin/positions/{id}/close` | Full/partial close |
| `PUT` | `/api/v1/portfolio/margin/positions/{id}/brackets` | Set/clear SL/TP |
| `GET` | `/api/v1/portfolio/margin/trades` | Margin trade history |

Snapshot fields on `GET /api/v1/portfolio`: `reservedMargin`, `marginLocked`, `marginUnrealizedPnL`, `marginEquity`, `marginPositions`.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/margin.go` |
| Store | `backend/internal/adapter/portfoliostore/margin.go` |
| Service | `backend/internal/service/portfolio/margin.go` |
| Worker | same interval as spot filler (`ProcessMarginMaintenance` in `filler.go`) |
| HTTP | `backend/internal/transport/http/handler/portfolio_margin.go` |

## Config

Uses existing `PORTFOLIO_DB_PATH` and `PORTFOLIO_ORDER_CHECK_INTERVAL`.

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/portfolio/ -run Margin -count=1
```

## Limitations

- Isolated margin only (no cross margin)
- No funding rates or fees on open/close
- One isolated position per open call (no size-add to existing id)
- Informational simulation — not real money / not financial advice
