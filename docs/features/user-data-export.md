# Feature: User data export

## Problem / goal

Users should be able to take their own product data offline: **watchlist**, **shares**, **price alert history**, **scanner backtest results**, and **paper portfolios**, as **JSON** or **CSV**. Large exports must not block the request forever — they run in the background with progress and cancel, only the owner can download the file, and files expire automatically.

## Behavior

| Capability | Detail |
|------------|--------|
| Formats | `json` (default) or `csv` (multi-section with `# section:…` markers) |
| Sections | `watchlist`, `shares`, `alerts`, `backtests`, `portfolios` (default: all) |
| Tenancy | Opaque `clientId` / `X-Client-Id` (same as watchlist) |
| Concurrency | At most **one** pending/running export per client (`409 conflict`) |
| Lifecycle | `pending` → `running` → `completed` \| `canceled` \| `failed` |
| Progress | `progressPct` 0–100 and `stage` (section name) |
| Download | `GET /api/v1/export/{id}/download` — **owner only** |
| TTL | Completed files deleted after `EXPORT_FILE_TTL` (default **1h**) |

### HTTP

| Method | Path | Notes |
|--------|------|--------|
| `POST` | `/api/v1/export` | Start job → **202** |
| `GET` | `/api/v1/export` | List jobs |
| `GET` | `/api/v1/export/{id}` | Status / progress |
| `POST` | `/api/v1/export/{id}/cancel` | Cancel pending/running |
| `GET` | `/api/v1/export/{id}/download` | File bytes when completed |

### Export contents

| Section | Source |
|---------|--------|
| watchlist | Owner's list items |
| shares | Shares granted by the owner |
| alerts | All price alerts for the client (active + triggered) |
| backtests | Backtest jobs + all signals (paged) |
| portfolios | Owned paper books: cash/balance, positions, trades, open orders, tax lots, recurring buys, margin positions/orders/trades, outgoing shares |

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/export.go` |
| Store | `backend/internal/adapter/exportstore/` |
| Service + worker | `backend/internal/service/export/` |
| HTTP | `backend/internal/transport/http/handler/export.go` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| MCP | `start_export`, `get_export`, `list_exports`, `cancel_export` |

## Config

| Env | Default |
|-----|---------|
| `EXPORT_DB_PATH` | `data/export.db` |
| `EXPORT_FILE_DIR` | `data/exports` |
| `EXPORT_FILE_TTL` | `1h` |
| `EXPORT_WORKER_INTERVAL` | `2s` |

## How to test

```bash
cd backend
go test ./internal/service/export/... ./internal/adapter/exportstore/... ./internal/transport/http/handler/ -count=1
```

Manual:

```bash
curl -X POST http://localhost:8080/api/v1/export \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"alice","format":"json"}'
# poll GET /api/v1/export/{id}?clientId=alice
# download GET /api/v1/export/{id}/download?clientId=alice
```

## Known limitations

- No real auth (clientId spoofing possible, same as other demo APIs).
- Export is always asynchronous (small jobs finish almost immediately).
- CSV is a single multi-section file, not a ZIP of sheets.
- Telegram has no export commands yet.
