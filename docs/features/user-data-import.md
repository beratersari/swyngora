# Feature: User data import

## Problem / goal

Users who exported watchlist, shares, alerts, backtests, and paper portfolios (JSON/CSV) should be able to **restore** that file. Before any write, the API shows how many records are valid, invalid, and would be added. Users choose **merge** (keep current data, skip duplicates) or **replace** (clear section data, then import). Large applies run in the background with progress and cancel.

## Behavior

| Step | What happens |
|------|----------------|
| 1. Preview | Upload file → parse/validate → `status=previewed` with section counts |
| 2. Confirm | Choose `merge` or `replace` → `pending` → worker applies |
| 3. Progress | Poll `progressPct` / `stage`; cancel anytime until terminal |

### Counts (per section)

| Field | Meaning |
|-------|---------|
| `valid` | Rows that pass validation |
| `invalid` | Rows that fail validation |
| `willAdd` | Valid rows that would be inserted under the selected mode |
| `duplicates` | Valid but already present (merge) or duplicate within the file |

### Modes

| Mode | Watchlist | Shares | Alerts | Backtests | Portfolios |
|------|-----------|--------|--------|-----------|------------|
| **merge** | Upsert by exchange+symbol; count new only | Skip existing grantee | Skip existing id | Skip existing id | Skip book if id or name already exists |
| **replace** | `Set` full list from file | Delete all shares then create | Delete all alerts then create | Delete all backtests then create | Delete all owned books then recreate from file |

- Ownership is always the **uploading** `clientId` (file `clientId` is ignored). The file's main paper book (`id` = original owner) is remapped to the importer; extra **UUID** books keep their ids. A non-UUID extra id (would occupy another tenant’s first-book `id = clientId`) or a UUID already used by another owner is reminted before insert.
- Trade/order/lot/margin ids are rekeyed on apply so two clients can share one database.
- Same symbol / share grantee / alert id / backtest id is never created twice.
- One **pending/running** import per client (`409`); multiple previews allowed until confirm conflicts with an active apply.

### HTTP

| Method | Path |
|--------|------|
| `POST` | `/api/v1/import/preview` — multipart `file` or raw JSON/CSV body |
| `POST` | `/api/v1/import/{id}/confirm` — `{ "mode": "merge"\|"replace" }` → 202 |
| `GET` | `/api/v1/import/{id}` — status + counts + progress |
| `GET` | `/api/v1/import` — list |
| `POST` | `/api/v1/import/{id}/cancel` | 

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/import.go` |
| Store | `backend/internal/adapter/importstore/` |
| Service | `backend/internal/service/dataimport/` |
| HTTP | `backend/internal/transport/http/handler/import.go` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| MCP | `preview_import`, `confirm_import`, `get_import`, `list_imports`, `cancel_import` |

## Config

| Env | Default |
|-----|---------|
| `IMPORT_DB_PATH` | `data/import.db` |
| `IMPORT_FILE_DIR` | `data/imports` |
| `IMPORT_FILE_TTL` | `1h` |
| `IMPORT_WORKER_INTERVAL` | `2s` |

## How to test

```bash
cd backend
go test ./internal/service/dataimport/... ./internal/adapter/importstore/... -count=1
```

## Known limitations

- MCP preview expects file content as a string (JSON best; large CSV may be awkward).
- Scanner **rules** are not restored (only backtest results/history).
- Paper portfolio restore does not include cash-movement history, equity snapshots, allocation baskets, or risk limits (balance/positions/trades/orders/lots/recurring/margin/shares are restored).
- No real auth (clientId model same as export).
