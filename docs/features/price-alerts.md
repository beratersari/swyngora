# Price alerts

## Goal

Price alerts support one-time/repeating modes, webhooks (immediate or hourly digest), and **quiet hours**.

## Quiet hours

| Setting | Meaning |
|---------|---------|
| `timeZone` | IANA name (default `UTC`) |
| `quietHours.enabled` | When true, defer delivery |
| `quietHours.start` / `end` | Local `HH:MM` (24h). If start > end, range crosses midnight (e.g. 22:00–08:00). |

- Fires still **create** pending notifications/digest items during quiet hours.
- `nextAttemptAt` is set to the **end of quiet hours** (not lost on restart).
- Retries also skip quiet windows.
- Digests still seal on the hour; POST waits until quiet hours end if needed.

### API example

```json
PUT /api/v1/alerts/webhook
{
  "url": "https://hooks.example.com/x",
  "deliveryMode": "immediate",
  "timeZone": "Europe/Istanbul",
  "quietHours": { "enabled": true, "start": "22:00", "end": "08:00" }
}
```

## Delivery modes

| `deliveryMode` | Behavior |
|----------------|----------|
| `immediate` | Each fire enqueued; POST after quiet hours allow |
| `hourly_digest` | UTC-hour batch; one POST after seal (+ quiet hours) |

## Alert modes

| `mode` | Behavior |
|--------|----------|
| `one_time` | Fire once → `triggered` |
| `repeating` | Edge-cross re-fire with re-arm |

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