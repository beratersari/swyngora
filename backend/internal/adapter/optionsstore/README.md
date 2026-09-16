# optionsstore

SQLite (and in-memory) store for **BTC/ETH options desk samples**.

The Deribit book summary is current-only. Open interest, put/call,
and dealer gamma (net GEX, regime, flip, index) are sampled every
few minutes while the backend runs. DVOL history still comes from
Deribit public REST.

## Layout

| File | Role |
|---|---|
| `sqlite.go` | Durable `options_samples` table |
| `memory.go` | In-process store for tests |

## How to test

```bash
cd backend && go test ./internal/adapter/optionsstore/ -count=1
```

## Config

| Env | Default |
|---|---|
| `OPTIONS_HISTORY_DB_PATH` | `data/options.db` |
| `OPTIONS_HISTORY_INTERVAL` | `5m` |
| `OPTIONS_HISTORY_RETENTION` | `14d` |

## Dependencies

Written by `GetOptions` and the options history worker. Read when
building `history` on `GET /api/v1/market/options`.
