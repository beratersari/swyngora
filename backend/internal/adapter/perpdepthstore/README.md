# perpdepthstore

SQLite (and in-memory) store for **market-wide perpetual bid/ask samples**.

Live depth is a snapshot. These rows let 15m / 1h / 4h windows compare
bid vs ask share of the liquid-perp book, and which coins moved that mix.

## Layout

| File | Role |
|---|---|
| `sqlite.go` | Durable `perp_depth_samples` table |
| `memory.go` | In-process store for tests |

## How to test

```bash
cd backend && go test ./internal/adapter/perpdepthstore/ -count=1
```
