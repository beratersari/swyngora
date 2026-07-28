# Alert store (`internal/adapter/alertstore`)

SQLite adapter for `domain.PriceAlertPort` (alerts, webhooks, immediate outbox, hourly digests).

## Schema

- `price_alerts` — alerts (`mode`, `armed`)
- `client_webhooks` — URL, `delivery_mode`, `timezone`, quiet hours (`quiet_enabled`, `quiet_start`, `quiet_end`)
- `alert_notifications` — immediate outbox (one row per fire)
- `alert_digests` / `alert_digest_items` — hourly batches; **PRIMARY KEY (digest_id, alert_id)** so each alert appears once per digest

## Tests

```bash
go test ./internal/adapter/alertstore/ -count=1
```