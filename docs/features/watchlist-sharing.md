# Feature: Watchlist sharing

## Problem / goal

Users want to collaborate on a market watchlist without sharing the same `clientId`.  
Owners can grant **viewer** or **editor** access to other client IDs, revoke access, and see an audit trail of who changed what and when.

## Behavior

| Role | Can view list | Add/remove symbols | Replace entire list | Manage shares | Delete list / revoke others |
|------|---------------|--------------------|---------------------|---------------|-----------------------------|
| **owner** | yes | yes | yes | yes | yes (owner identity) |
| **editor** | yes | yes | no | no | no |
| **viewer** | yes | no | no | no | no |

Rules:

- Tenancy remains opaque `clientId` / `X-Client-Id` (no server auth yet).
- Same list cannot be shared with the same grantee twice (`400`); use `PATCH` to change role.
- Cannot share with yourself.
- Max **50** shares per owner list.
- Every share grant/update/revoke and every item add/remove/replace is recorded in the audit log with `actorClientId` + `createdAt`.

### HTTP

| Method | Path | Who | Notes |
|--------|------|-----|--------|
| `GET` | `/api/v1/watchlist?ownerClientId=` | owner / viewer / editor | Omit `ownerClientId` for own list; response includes `role` |
| `POST` | `/api/v1/watchlist/items` | owner / editor | Body may include `ownerClientId` |
| `DELETE` | `/api/v1/watchlist/items` | owner / editor | Query may include `ownerClientId` |
| `PUT` | `/api/v1/watchlist` | **owner only** | Full replace |
| `POST` | `/api/v1/watchlist/shares` | **owner only** | `{ granteeClientId, role: viewer\|editor }` |
| `PATCH` | `/api/v1/watchlist/shares` | **owner only** | Change role |
| `GET` | `/api/v1/watchlist/shares` | owner | List outgoing shares |
| `DELETE` | `/api/v1/watchlist/shares?granteeClientId=` | **owner only** | Revoke |
| `GET` | `/api/v1/watchlist/shared` | grantee | Lists shared *with me* |
| `GET` | `/api/v1/watchlist/audit` | owner | Change history |

Persistence: same SQLite DB as watchlists (`WATCHLIST_DB_PATH`) — tables `watchlist_shares`, `watchlist_audit`.

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/watchlist.go` |
| Store | `backend/internal/adapter/watchliststore/{memory,sqlite}.go` |
| Service | `backend/internal/service/watchlist/service.go` |
| HTTP | `backend/internal/transport/http/handler/watchlist.go` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| MCP | `backend/internal/transport/mcp` |
| AI tools | `ai/src/swyngora_ai/tools/market_http.py` |

## How to test / verify

```bash
cd backend
go test ./internal/service/watchlist/... ./internal/adapter/watchliststore/... ./internal/transport/http/handler/... ./internal/transport/mcp/...
```

Manual:

```bash
# owner shares with editor
curl -X POST http://localhost:8080/api/v1/watchlist/shares \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"alice","granteeClientId":"bob","role":"editor"}'

# bob adds a symbol to alice's list
curl -X POST http://localhost:8080/api/v1/watchlist/items \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"bob","ownerClientId":"alice","exchange":"binance","symbol":"ETHUSDT"}'

# alice sees audit
curl 'http://localhost:8080/api/v1/watchlist/audit?clientId=alice'
```

## MCP / AI tools

`share_watchlist`, `update_watchlist_share`, `revoke_watchlist_share`, `list_watchlist_shares`, `list_shared_watchlists`, `list_watchlist_audit`;  
`get_watchlist` / `add_watchlist_item` / `remove_watchlist_item` accept optional `ownerClientId` / `owner_client_id`.

## Known limitations / follow-ups

- No real authentication — anyone who knows a `clientId` can act as that client.
- No multi-list-per-user model yet (one list per `clientId`).
- UI for sharing is not in this change (API + MCP + AI tools only).
- Telegram bot still operates only on the caller's own `tg-<user_id>` list.
