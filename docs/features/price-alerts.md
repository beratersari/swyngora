# Price alerts

## Goal

Let users create **one-shot** alerts when a symbol's last price goes **above** or **below** a target. Alerts survive process restarts and are never re-fired after triggering. Optional webhooks deliver a durable, at-most-once notification when an alert triggers.

## Behavior

1. Client creates an alert with `clientId`, `exchange`, `symbol`, `condition` (`above`|`below`), `targetPrice`.
2. Status starts as `active`.
3. Background checker (default every 30s) loads active alerts, fetches last price (deduped per exchange+symbol), and when the threshold is met runs `MarkTriggered` once.
4. Status becomes `triggered` with `triggeredAt` and `triggeredPrice`. Further market moves do not re-open or re-fire the alert.
5. If the client has a webhook URL, a **notification outbox row** is enqueued (unique on `alertId` — at most one notification per alert).
6. A background deliverer POSTs the JSON payload to the webhook. Failures are retried with exponential backoff; pending rows survive process restarts. Successful delivery sets status `delivered` and is never re-sent.
7. Clients can list, get, or delete alerts; get/set/clear webhook URL at any time.

Informational only — not financial advice.

## Webhook payload (example)

```json
{
  "type": "price_alert.triggered",
  "alertId": "…",
  "clientId": "web-abc",
  "exchange": "binance",
  "symbol": "BTCUSDT",
  "condition": "above",
  "targetPrice": 100000,
  "lastPrice": 100500.25,
  "triggeredAt": "2026-07-28T12:00:00Z",
  "status": "triggered",
  "note": "Informational only — not financial advice."
}
```

## API

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/v1/alerts` | List for `clientId` / `X-Client-Id` |
| `POST` | `/api/v1/alerts` | Create |
| `GET` | `/api/v1/alerts/{id}` | Get one |
| `DELETE` | `/api/v1/alerts/{id}` | Delete |
| `GET` | `/api/v1/alerts/webhook` | Get webhook URL |
| `PUT` | `/api/v1/alerts/webhook` | Set webhook URL (`url` absolute http/https) |
| `DELETE` | `/api/v1/alerts/webhook` | Clear webhook URL |

OpenAPI: `backend/api/openapi/openapi.yaml`.

## Code

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/alert.go` |
| Store | `backend/internal/adapter/alertstore` (alerts + webhooks + outbox) |
| Service + checker + deliverer | `backend/internal/service/pricealert` |
| HTTP | `backend/internal/transport/http/handler/alerts.go` |
| MCP | `list_price_alerts`, `create_price_alert`, `delete_price_alert`, webhook tools |

## Config

| Env | Default |
|-----|---------|
| `ALERTS_DB_PATH` | `data/alerts.db` |
| `ALERT_CHECK_INTERVAL` | `30s` |
| `WEBHOOK_DELIVERY_INTERVAL` | `5s` |
| `WEBHOOK_HTTP_TIMEOUT` | `10s` |
| `WEBHOOK_MAX_ATTEMPTS` | `8` |

## Tests

```bash
cd backend
go test ./internal/domain/ -run Alert -count=1
go test ./internal/adapter/alertstore/ ./internal/service/pricealert/ ./internal/transport/http/handler/ -run "Alert|Webhook|Deliverer|Notification" -count=1
```

## Limits

- Max **50** alerts per `clientId` (any status).
- At most **one** webhook notification per alert (unique `alert_id` in outbox).
- Same tenancy model as watchlist (opaque `clientId`, no server auth yet).