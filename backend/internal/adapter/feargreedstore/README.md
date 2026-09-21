# Fear and greed history

SQLite samples of Swyngora's own 0–100 fear and greed score.

## Layout

| File | Role |
|---|---|
| `sqlite.go` | Durable store (`data/feargreed.db` by default) |
| `memory.go` | In-process store for tests |

One row per 15-minute bucket: score, level, and which input pulled the score the most. The service writes a sample when it publishes a score. A background worker also samples on `FEAR_GREED_HISTORY_INTERVAL` (default 15m) and deletes rows older than `FEAR_GREED_HISTORY_RETENTION` (default 90 days).

## Test

```bash
cd backend && go test ./internal/adapter/feargreedstore/
```
