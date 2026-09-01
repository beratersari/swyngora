# Feature: Multi-device watchlist sync

## Problem / goal

When the same `clientId` is used on two devices, last-write-wins overwrites the other device’s symbols.  
Swyngora keeps a **version** per list, auto-merges non-overlapping adds, and returns a **conflict** when the same symbol was deleted on one side and changed on the other so the user can choose.

## Behavior

| Field | Meaning |
|-------|---------|
| `version` | Monotonic integer on every successful mutation (starts at **0** for empty lists) |
| `baseVersion` | Version the client last loaded (from GET) |

### Write rules

1. Client GETs list → stores `version` + items.
2. Client mutates with `baseVersion` equal to that value.
3. If `baseVersion` matches server → apply, `version++`.
4. If versions differ:
   - **Different symbols added** on both sides → **auto-merge** (union) and save.
   - **Same symbol, different notes** → **409** `update_vs_update`.
   - **Client deletes, server still has (changed) item** → **409** `delete_vs_update`.
   - **Client keeps/updates, server deleted** (with `baseItems`) → **409** `update_vs_delete`.
   - **Client deletes, server unchanged from base** → accept delete.

### Resolving conflicts

No separate resolve endpoint. After a **409**:

1. Show `conflict.server` vs `conflict.clientItem` / `serverItem` per row.
2. Let the user pick keep server / keep client / delete for each conflict.
3. Build the full desired `items` list (start from `autoMerged` if present).
4. `PUT /api/v1/watchlist` with `baseVersion: conflict.serverVersion` and resolved `items`.

### API fields

| Operation | How to send version |
|-----------|---------------------|
| GET | Response includes `version` |
| PUT replace | Body `baseVersion`, optional `baseItems` |
| POST add | Body `baseVersion` |
| DELETE item | Query `baseVersion` or header `If-Match` |

Omitting `baseVersion` keeps **unconditional** writes (legacy clients / Telegram / import). The product web app always sends `baseVersion` from the last GET.

## Where the code lives

| Layer | Path |
|-------|------|
| Domain merge rules | `backend/internal/domain/watchlist_sync.go` |
| Store CAS | `watchliststore` `Set/Add/Remove(..., expectedVersion)` |
| Service | `backend/internal/service/watchlist/service.go` |
| HTTP 409 body | `backend/internal/transport/http/handler/watchlist.go` |

## Tests

```bash
cd backend
go test ./internal/domain/ ./internal/service/watchlist/ ./internal/adapter/watchliststore/ -count=1
```

## Known limitations

- True 3-way delete detection on full replace is best with `baseItems` from the last GET.
- Without `baseVersion`, concurrent devices can still overwrite (legacy mode).
