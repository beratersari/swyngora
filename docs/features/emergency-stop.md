# Feature: Paper emergency stop

## Problem / goal

A daily loss limit can block new buys and still leave positions, resting orders, and recurring buys working. When the day's trading loss passes a percent the user chooses, the book should flatten itself, and the user should see which step worked and which failed.

## Behavior

The stop is off until it is turned on. `lossPct` of 5 means 5 percent. The loss is today's trading result against the UTC day baseline, the same figure as the daily loss limit. Deposits, withdrawals, and transfers are not part of it.

While the stop is on and that loss is past the line, each pass:

1. Cancels open and pending paper orders.
2. Cancels open spread orders.
3. Pauses active recurring buys.
4. Sells each open spot position, including equities, at the paper market.
5. Closes each open margin position.

A step that fails is stored with the reason and tried again on the next pass while the loss is still past the line. A step that worked is not repeated. Turning the stop off stops further tries and does not buy back what was closed. The check runs when the switch is saved and about every 15 seconds with the paper order filler.

The risk-limit brakes still never close a position. This stop is separate.

## API

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/portfolio/emergency-stop` | Switch, today's loss, last trip |
| `PUT` | `/api/v1/portfolio/emergency-stop` | `{ "enabled": true, "lossPct": 5 }` |

`run.actions[]` has `kind`, `targetId`, `symbol`, `status` (`ok` or `failed`), and `detail` when it failed. Kinds are `cancel_order`, `cancel_spread`, `pause_recurring`, `close_spot`, `close_margin`, and `cancel_pending` when the whole cancel fails.

MCP `get_emergency_stop` and `set_emergency_stop`. The portfolio page shows the switch under the exposure limit.

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/emergency_stop.go` |
| Store | `emergency_stops` and `emergency_stop_runs` in `portfoliostore` |
| Service | `backend/internal/service/portfolio/emergency_stop.go` |
| Worker | `OrderFiller.RunOnce` calls `ProcessEmergencyStops` before new fills |
| HTTP | `backend/internal/transport/http/handler/portfolio_emergency.go` |
| UI | `frontend/src/components/organisms/EmergencyStop` |

## Tests

```bash
cd backend
go test ./internal/service/portfolio/ -run TestEmergencyStop -count=1
```

## Known limits

The stop setting is not in the paper-book export. A sell the visible book cannot hold stays failed until a later pass can fill it. The loss line resets on the next UTC day, the same way the daily loss limit does.
