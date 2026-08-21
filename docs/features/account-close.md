# Feature: Account close and reopen

## Problem / goal

Users need to close their account (opaque `clientId` tenancy). While closed they cannot use product APIs as that user, and **shared watchlists they own stop being accessible**. Data is retained for **7 days** so they can reopen; after that it is purged.

## Behavior

| State | API as that clientId | Shared lists they own | Data |
|-------|----------------------|------------------------|------|
| **active** | normal | shared | kept |
| **closed** (≤7 days) | blocked (403 `account_closed`) except status/reopen/close; Telegram tenant commands also refuse | **no access** for grantees | kept; active jobs canceled; paper recurring plans paused and open paper/margin orders canceled; workers skip the tenant |
| **after 7 days** | N/A | N/A | **purged** |

### HTTP

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/v1/account` | Status (`active` / `closed`), `purgeAt`, `canReopen` |
| `POST` | `/api/v1/account/close` | Close account; sets `purgeAt = now + 7d` |
| `POST` | `/api/v1/account/reopen` | Restore access if still before `purgeAt` |

Middleware reads tenant id from `X-Client-Id`, `?clientId=`, or JSON `clientId` (body is restored for the handler). User-scoped routes **fail closed** with `400 invalid_argument` when no clientId is present. Closed accounts get `403 account_closed`. Public market routes are unaffected. **`/mcp` is not header-gated** (tools pass `clientId` in the JSON body); in-process MCP tools call `RequireActive` when `clientId` is present so closed accounts cannot mutate via agents.

Reserved tenant names (`default`, `anonymous`, `http-default`, `ai-assistant`, enumerable `tg-<digits>`) are rejected. Telegram users get a persisted UUID mapping (not `tg-<userId>`).

### Purge contents (after grace)

- Watchlist items + meta, shares (as owner or grantee), audit  
- Price alerts, webhooks, notification/digest rows  
- Scanner rules, results, backtests (+ signals)  
- Export jobs + files  
- Import jobs + source/payload files  
- User API keys  
- Paper books (positions, orders, recurring plans, margin)  
- Price-diff watches  
- Account row removed  

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/account.go` |
| Store | `backend/internal/adapter/accountstore/` |
| Service + worker | `backend/internal/service/account/` |
| Middleware | `backend/internal/transport/http/middleware/account.go` |
| HTTP | `backend/internal/transport/http/handler/account.go` |
| Telegram | `backend/internal/transport/telegram/commands.go` (`clientIDForUser` → `RequireActive`) |

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
- REST `AccountGate` reads `X-Client-Id`, `?clientId=`, then JSON `clientId` (body is restored).  
- MCP tools with `clientId` are blocked while closed (same `RequireActive` error as REST).  
- Telegram uses the same `RequireActive` check after resolving the mapped tenant id. Public market commands (`/price`, `/rsi`, …) stay available.  
- Paper books are frozen on close (plans paused, open orders canceled) and purged after grace.  
- Background workers skip closed tenants: price-alert checker, webhook deliverer, scanner `RunOnce`, price-diff `ProcessActiveWatches`. Rows stay until purge so reopen works.
