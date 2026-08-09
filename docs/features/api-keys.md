# Feature: Per-account API keys

## Problem / goal

A single process `API_AUTH_TOKEN` is too coarse for bots and extra apps. Users need **named keys** on their own `clientId` that can be **read-only** or allowed to **trade**, listed, and revoked without sharing full account access.

## Behavior

| Permission | Allowed |
|------------|---------|
| `read` | GET (and HEAD) on tenant APIs; `POST /api/v1/ai/chat` |
| `trade` | `read` plus mutations (paper orders, cash, watchlist writes, alerts, …) |

User keys **cannot**:

- Create, list, or revoke other keys
- Close or reopen the account
- Use `/mcp` unless `permission=trade`

The process master token (`API_AUTH_TOKEN`) still has full access and is how the main app manages keys when the API is locked down. In open local mode (empty master token), `X-Client-Id` is enough to manage keys; a `swy_…` user key still binds that client and its scopes.

Secrets are stored as SHA-256 hashes. The full token is returned **once** on create (`swy_` + 48 hex chars). List shows only `prefix` (first 12 characters).

Limits: 20 active keys per `clientId`; name 1–64 characters.

### HTTP

| Method | Path | Notes |
|--------|------|--------|
| `POST` | `/api/v1/account/api-keys` | `{ name, permission? }` → includes `secret` once |
| `GET` | `/api/v1/account/api-keys` | Metadata only |
| `DELETE` | `/api/v1/account/api-keys/{id}` | Revoke immediately |

Send the secret as `Authorization: Bearer <secret>` or `X-API-Key: <secret>`. The key’s `clientId` is used (header spoofing is ignored).

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/apikey.go` |
| Store | `backend/internal/adapter/accountstore/` (`api_keys` table) |
| Service | `backend/internal/service/apikey/` |
| Middleware | `backend/internal/transport/http/middleware/apiauth.go` |
| HTTP | `backend/internal/transport/http/handler/apikeys.go` |

## Tests

```bash
cd backend
go test ./internal/service/apikey/... ./internal/adapter/accountstore/... ./internal/transport/http/middleware/... -count=1
```

## Known limitations

- `clientId` is still not a login; master token + `X-Client-Id` is the “main account”
- Read vs trade is path/method based, not per-tool MCP filtering (MCP requires `trade`)
- Account purge deletes all keys for that client
