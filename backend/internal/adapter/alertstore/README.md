# Alert store (`internal/adapter/alertstore`)

SQLite adapter for `domain.PriceAlertPort`.

## Schema

`price_alerts` — id, client_id, exchange, symbol, condition (`above`|`below`), target_price, status (`active`|`triggered`), created_at, triggered_at, triggered_price.

`MarkTriggered` uses `UPDATE … WHERE status = 'active'` so each alert fires **at most once**.

## Config

| Env | Default |
|-----|---------|
| `ALERTS_DB_PATH` | `data/alerts.db` |

## Tests

```bash
go test ./internal/adapter/alertstore/ -count=1
```