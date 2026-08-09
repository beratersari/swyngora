# Feature: Paper portfolio sharing

## Problem / goal

Users want to let a friend or bot **see** a paper book, or **trade** on it, without handing over the owner `clientId`. Depositing virtual cash, deleting the book, and changing who has access stay with the owner.

## Behavior

| Role | Snapshot, positions, trades, performance | Place / amend / cancel orders (incl. margin, DCA, baskets) | Deposit / withdraw / transfer | Delete book | Share / unshare / change role |
|------|------------------------------------------|-------------------------------------------------------------|-------------------------------|-------------|-------------------------------|
| **owner** | yes | yes | yes | yes | yes |
| **trader** | yes | yes | no | no | no |
| **viewer** | yes | no | no | no | no |

Rules:

- Share is **per book**, not per account.
- Cannot share with yourself.
- Same grantee cannot be shared twice (`400`); use `PATCH` to change role.
- Max **50** shares per book.
- Grantee selects the book with `portfolioId` (UUID) or `ownerClientId` + name.

## HTTP

| Method | Path | Who |
|--------|------|-----|
| `POST` | `/api/v1/portfolio/shares` | owner `{ granteeClientId, role, portfolioId? }` |
| `PATCH` | `/api/v1/portfolio/shares` | owner change role |
| `GET` | `/api/v1/portfolio/shares?portfolioId=` | owner outgoing |
| `DELETE` | `/api/v1/portfolio/shares?granteeClientId=` | owner revoke |
| `GET` | `/api/v1/portfolios/shared` | grantee incoming |
| `GET` | `/api/v1/portfolio?portfolioId=` | owner / viewer / trader |
| `GET` | `/api/v1/ws` `subscribe_portfolio` | same as GET portfolio (see [`realtime.md`](realtime.md)) |

## Where the code lives

| Layer | Path |
|-------|------|
| Domain | `backend/internal/domain/portfolio.go` |
| Store | `backend/internal/adapter/portfoliostore/shares.go` |
| Service | `backend/internal/service/portfolio/{access,share}.go` |
| HTTP | `backend/internal/transport/http/handler/portfolio_share.go` |
| OpenAPI | `backend/api/openapi/openapi.yaml` |
| MCP | `share_portfolio`, `update_portfolio_share`, `revoke_portfolio_share`, `list_portfolio_shares`, `list_shared_portfolios` |
| Telegram | `/portfolio share`, `/portfolio unshare`, `/portfolio shares`, `/portfolio shared` |

## How to test / verify

```bash
cd backend
go test ./internal/service/portfolio/ ./internal/adapter/portfoliostore/ ./internal/transport/http/handler/ ./internal/transport/mcp/ ./internal/transport/telegram/
```

## Known limitations / follow-ups

- No real authentication — anyone who knows a `clientId` can act as that client.
- Account purge does not yet wipe portfolio shares (book delete does).
- Product UI for sharing is API + MCP + Telegram only in this change.
