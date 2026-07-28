# Price alerts

## Goal

Let users create **one-time** or **repeating** alerts when a symbol's last price goes **above** or **below** a target. Alerts and webhook outbox rows survive process restarts.

## Modes

| Mode | Behavior |
|------|----------|
| `one_time` (default) | Fires once when the condition is met; status becomes `triggered` and never re-fires. |
| `repeating` | Stays `active`. Fires on each **edge** into the condition zone. While price stays on that side it does not re-fire. When price returns to the **safe** side the alert re-arms; the next cross fires again. |

Repeating starts **disarmed** so a price already on the trigger side at create/first poll does not fire until it leaves and re-crosses.

## Behavior

1. Client creates an alert with `clientId`, `exchange`, `symbol`, `condition`, `targetPrice`, optional `mode`.
2. Background checker (default every 30s) evaluates active alerts against last price.
3. On fire: update last `triggeredAt` / `triggeredPrice`; one_time → status `triggered`; repeating → stay `active`, set `armed=false`.
4. If a webhook URL is set, enqueue a durable notification row (new row per fire) and deliver in the background with retries.

Informational only — not financial advice.

## API

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/v1/alerts` | List for `clientId` / `X-Client-Id` |
| `POST` | `/api/v1/alerts` | Create (`mode`: `one_time` \| `repeating`) |
| `GET` | `/api/v1/alerts/{id}` | Get one |
| `DELETE` | `/api/v1/alerts/{id}` | Delete |
| `GET`/`PUT`/`DELETE` | `/api/v1/alerts/webhook` | Webhook URL |

OpenAPI: `backend/api/openapi/openapi.yaml`.

## Code

| Layer | Path |
|-------|------|
| Domain | `EvaluateAlert` edge logic in `domain/alert.go` |
| Store | `adapter/alertstore` (`mode`, `armed`, multi-row outbox) |
| Service | `ProcessPrice` + checker |
| HTTP / MCP / AI | create accepts `mode` |

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
go test ./internal/domain/ -run 'Alert|Evaluate' -count=1
go test ./internal/service/pricealert/ -run 'Repeating|Checker|Create' -count=1
go test ./internal/adapter/alertstore/ ./internal/transport/http/handler/ -count=1
```

## Limits

- Max **50** alerts per `clientId`.
- Same tenancy model as watchlist (opaque `clientId`).