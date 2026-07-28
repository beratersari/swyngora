# Price alerts

## Goal

Let users create **one-shot** alerts when a symbol's last price goes **above** or **below** a target. Alerts survive process restarts and are never re-fired after triggering.

## Behavior

1. Client creates an alert with `clientId`, `exchange`, `symbol`, `condition` (`above`|`below`), `targetPrice`.
2. Status starts as `active`.
3. Background checker (default every 30s) loads active alerts, fetches last price (deduped per exchange+symbol), and when the threshold is met runs `MarkTriggered` once.
4. Status becomes `triggered` with `triggeredAt` and `triggeredPrice`. Further market moves do not re-open or re-fire the alert.
5. Clients can list, get, or delete alerts at any time.

Informational only — not financial advice.

## API

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/v1/alerts` | List for `clientId` / `X-Client-Id` |
| `POST` | `/api/v1/alerts` | Create |
| `GET` | `/api/v1/alerts/{id}` | Get one |
| `DELETE` | `/api/v1/alerts/{id}` | Delete |

OpenAPI: `backend/api/openapi/openapi.yaml`.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/alert.go` |
| Store | `backend/internal/adapter/alertstore` |
| Service + checker | `backend/internal/service/pricealert` |
| HTTP | `backend/internal/transport/http/handler/alerts.go` |
| MCP | `list_price_alerts`, `create_price_alert`, `delete_price_alert` |

## Config

| Env | Default |
|-----|---------|
| `ALERTS_DB_PATH` | `data/alerts.db` |
| `ALERT_CHECK_INTERVAL` | `30s` |

## Tests

```bash
cd backend
go test ./internal/domain/ -run Alert -count=1
go test ./internal/adapter/alertstore/ ./internal/service/pricealert/ ./internal/transport/http/handler/ -run Alert -count=1
```

## Limits

- Max **50** alerts per `clientId` (any status).
- Same tenancy model as watchlist (opaque `clientId`, no server auth yet).