# Alert store (`internal/adapter/alertstore`)

SQLite adapter for `domain.PriceAlertPort` (alerts, webhooks, notification outbox).

## Schema

- `price_alerts` — alert rows; `MarkTriggered` is one-shot (`WHERE status = 'active'`).
- `client_webhooks` — one webhook URL per `client_id`.
- `alert_notifications` — durable outbox; **unique `alert_id`** so each alert enqueues **at most one** notification. Status: `pending` → `delivered` | `failed`.

## Config

| Env | Default |
|-----|---------|
| `ALERTS_DB_PATH` | `data/alerts.db` |
| `WEBHOOK_DELIVERY_INTERVAL` | `5s` |
| `WEBHOOK_MAX_ATTEMPTS` | `8` |

## Tests

```bash
go test ./internal/adapter/alertstore/ -count=1
```