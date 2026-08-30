# Feature: Per-account API keys

## Problem / goal

A single process `API_AUTH_TOKEN` is too coarse for bots and extra apps. Users need **named keys** on their own `clientId` that can be **read-only** or allowed to **trade**, listed, and revoked without sharing full account access.

## Behavior

| Permission | Allowed |
|------------|---------|
| `read` | GET (and HEAD) on tenant APIs; `POST /api/v1/ai/chat` (and stream) with **read-only tools** |
| `trade` | `read` plus mutations (paper orders, cash, watchlist writes, alerts, …); AI chat may use those tools |

User keys **cannot**:

- Create, list, or revoke other keys (including via AI tools — the proxy passes `canManageKeys=false`)
- Close or reopen the account
- Use `/mcp` unless `permission=trade`
- Use AI to place trades or mutate state when `permission=read` (`canTrade=false` is enforced in the AI tool HTTP layer even though tools authenticate with the process master token)

The process master token (`API_AUTH_TOKEN`) can manage keys and, **from loopback only**, act as any `X-Client-Id` (local Vite, AI HTTP tools). Remote clients cannot impersonate a tenant unless `ALLOW_MASTER_IMPERSONATE=true`. They may mint the first key for a client that has none. In open local mode (empty master token), `X-Client-Id` is enough to manage keys; a `swy_…` user key still binds that client and its scopes.

The web Settings desk shows the secret once on create. It does **not** install a `permission=read` secret as the browser session token (that would 403 every paper mutation). A newly created `trade` key may be stored in `swyngora.apiAuthToken`.

Secrets are stored as SHA-256 hashes. The full token is returned **once** on create (`swy_` + 48 hex chars). List shows only `prefix` (first 12 characters).

Limits: 20 active keys per `clientId`; name 1–64 characters. `clientId` is validated with `domain.NormalizeClientID` (reserved names and enumerable `tg-<digits>` are rejected).

### HTTP

| Method | Path | Notes |
|--------|------|--------|
| `POST` | `/api/v1/account/api-keys` | `{ name, permission? }` → includes `secret` once |
| `GET` | `/api/v1/account/api-keys` | Metadata only |
| `DELETE` | `/api/v1/account/api-keys/{id}` | Revoke immediately |

Send the secret as `Authorization: Bearer <secret>` or `X-API-Key: <secret>`. The key’s `clientId` is used (header spoofing is ignored). Do **not** put the secret on a query string. Browsers mint `POST /api/v1/realtime/ticket` and pass `?ticket=` on the WebSocket only.

The process master token cannot act as an arbitrary tenant from a remote client (`ALLOW_MASTER_IMPERSONATE=false` by default). Loopback (local Vite proxy, AI HTTP tools) may still impersonate. Remote master may create the **first** key for a client that has none; it cannot mint keys for an existing tenant or read that tenant’s data.

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

- `clientId` is still not a login; user keys are the tenant credential. Remote master impersonation is off unless `ALLOW_MASTER_IMPERSONATE=true`
- Read vs trade is path/method based, not per-tool MCP filtering (MCP requires `trade`)
- Account purge deletes all keys for that client
