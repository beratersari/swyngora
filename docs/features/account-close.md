# Feature: Account close and reopen

## Problem / goal

Users need to close their account (opaque `clientId` tenancy). While closed they cannot use product APIs as that user, and **shared watchlists they own stop being accessible**. Data is retained for **7 days** so they can reopen; after that it is purged.

## Behavior

| State | API as that clientId | Shared lists they own | Data |
|-------|----------------------|------------------------|------|
| **active** | normal | shared | kept |
| **closed** (≤7 days) | blocked (403 `account_closed`) except status/reopen/close | **no access** for grantees | kept; active jobs canceled |
| **after 7 days** | N/A | N/A | **purged** |

### HTTP

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/v1/account` | Status (`active` / `closed`), `purgeAt`, `canReopen` |
| `POST` | `/api/v1/account/close` | Close account; sets `purgeAt = now + 7d` |
| `POST` | `/api/v1/account/reopen` | Restore access if still before `purgeAt` |

Middleware blocks closed `X-Client-Id` / query `clientId` on other user-scoped REST routes. Public market routes are unaffected. **`/mcp` is not header-gated** (tools pass `clientId` in the JSON body); in-process MCP tools call `RequireActive` when `clientId` is present so closed accounts cannot mutate via agents.

### Purge contents (after grace)

- Watchlist items + meta, shares (as owner or grantee), audit  
- Price alerts, webhooks, notification/digest rows  
- Scanner rules, results, backtests (+ signals)  
- Export jobs + files  
- Import jobs + source/payload files  
- Account row removed  

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/account.go` |
| Store | `backend/internal/adapter/accountstore/` |
| Service + worker | `backend/internal/service/account/` |
| Middleware | `backend/internal/transport/http/middleware/account.go` |
| HTTP | `backend/internal/transport/http/handler/account.go` |

## Config

| Env | Default |
|-----|---------|
| `ACCOUNT_DB_PATH` | `data/accounts.db` |
| `ACCOUNT_PURGE_INTERVAL` | `1h` |

Grace duration is fixed at **7 days** (`domain.AccountCloseGrace`).

## Tests

```bash
cd backend
go test ./internal/service/account/... -count=1
```

## Known limitations

- “Login” is the clientId model; there is no separate auth provider yet.  
- Body-only `clientId` (no header/query) is not gated by REST middleware. Prefer `X-Client-Id`.  
- MCP tools with `clientId` are blocked while closed (same `RequireActive` error as REST).  
- Paper portfolio purge is not included in this change.
