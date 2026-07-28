# Price alerts

## Goal

Let users create **one-time** or **repeating** alerts when a symbol's last price goes **above** or **below** a target. Alerts and notification outboxes survive process restarts. Webhooks can be delivered **immediately** or as an **hourly digest**.

## Modes

| `mode` | Behavior |
|--------|----------|
| `one_time` (default) | Fires once when the condition is met; status becomes `triggered`. |
| `repeating` | Stays `active`. Fires on each edge into the condition zone; re-arms on the safe side. |

## Webhook delivery

| `deliveryMode` | Behavior |
|----------------|----------|
| `immediate` (default) | Each fire enqueues a pending notification and is POSTed soon with retries. |
| `hourly_digest` | Fires in the same UTC hour are collected; after the hour ends, one webhook POST is sent with all distinct alerts in that window. The same `alertId` appears **at most once** per digest (latest payload wins). Digests are durable, retry on failure, and survive restarts. |

### Digest payload (example)

```json
{
  "type": "price_alert.digest",
  "digestId": "…",
  "clientId": "web-abc",
  "windowStart": "2026-07-28T14:00:00Z",
  "windowEnd": "2026-07-28T15:00:00Z",
  "count": 2,
  "alerts": [ { "type": "price_alert.triggered", "alertId": "…", "…" : "…" } ],
  "note": "Informational only — not financial advice."
}
```

## API

| Method | Path | Notes |
|--------|------|--------|
| `POST` | `/api/v1/alerts` | Create (`mode`: `one_time` \| `repeating`) |
| `GET`/`DELETE` | `/api/v1/alerts`, `…/{id}` | List / get / delete |
| `PUT` | `/api/v1/alerts/webhook` | Set `url` + optional `deliveryMode` |
| `GET`/`DELETE` | `/api/v1/alerts/webhook` | Get / clear webhook |

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
go test ./internal/domain/ ./internal/adapter/alertstore/ ./internal/service/pricealert/ -count=1
```