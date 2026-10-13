# Alert store (`internal/adapter/alertstore`)

SQLite adapter for `domain.PriceAlertPort` (alerts, webhooks, immediate outbox, hourly digests).

## Schema

- `price_alerts` — alerts (`kind`, `range_pct`, `mode`, `armed`, `situation_fire_json`). The last situation fire stays through re-arm and is replaced by the next fire. Empty or invalid JSON leaves the fire off
- `client_webhooks` — URL, `delivery_mode`, `timezone`, quiet hours (`quiet_enabled`, `quiet_start`, `quiet_end`)
- `alert_notifications` — immediate outbox (one row per fire)
- `alert_digests` / `alert_digest_items` — hourly batches; **PRIMARY KEY (digest_id, alert_id)** so each alert appears once per digest

## Tests

```bash
go test ./internal/adapter/alertstore/ -count=1
```