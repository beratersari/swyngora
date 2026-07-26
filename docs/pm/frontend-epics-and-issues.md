# GitLab project management: Frontend epics & issues

**Project:** `trace-analysis/swyngora`  
**Host:** https://nova.teachx.ai  
**Labels (create if missing):** `frontend`, `type::epic`, `type::feature`, `type::chore`, `priority::p0`, `priority::p1`

This file is the **source of truth** for creating GitLab epics/issues when MCP or API credentials are available.

**MCP setup:** see [`gitlab-mcp-setup.md`](./gitlab-mcp-setup.md).

**Local board (current):** [`project-management/`](../../project-management/) — use this for day-to-day task status until GitLab issues are created.

---

## Epic 1 — Frontend project initialization (FIRST)

| Field | Value |
|---|---|
| **Title** | `[frontend] Project initialization` |
| **Type** | Epic |
| **Labels** | `frontend`, `priority::p0` |
| **Description** | See below |
| **Design** | `docs/design/frontend-project-initialization.md` |
| **System design** | `docs/design/frontend-system-design.md` |

### Epic description (paste into GitLab)

```markdown
## Summary

Initialize the production React web app under `frontend/` so feature work can start.

This is **P0** and **blocks** multi-exchange spot markets UI.

## Goals

- Vite + React + TypeScript scaffold
- Lint / test tooling
- Atomic Design + feature folder structure
- Redux Toolkit + RTK Query `baseApi`
- OpenAPI codegen from `backend/api/openapi/openapi.yaml`
- App shell + router + env (`VITE_API_BASE_URL`)
- Package README + AGENTS.md

## Out of scope

Markets table UI, watchlist, charts, auth.

## Design docs

- `docs/design/frontend-project-initialization.md`
- `docs/design/frontend-system-design.md`

## Acceptance

- `npm install && npm run dev` works in `frontend/`
- `npm run codegen:api` works
- Placeholder Markets route renders
- Structure matches AGENTS.md §6.8 / §6.9

## Child issues

See INIT-1 … INIT-7.
```

### Child issues — Epic 1

#### INIT-1 — Scaffold Vite React TypeScript app

```markdown
## Summary
Create Vite + React + TypeScript project under `frontend/` with package.json and entrypoints.

## Tasks
- [ ] Vite React-TS template (or equivalent manual scaffold)
- [ ] `index.html`, `src/main.tsx`
- [ ] npm lockfile committed
- [ ] `.env.example` with `VITE_API_BASE_URL=http://localhost:8080`

## Acceptance
`npm install && npm run dev` serves a blank app.

## Labels
`frontend`, `type::chore`, `priority::p0`

## Design
docs/design/frontend-project-initialization.md §5.1
```

#### INIT-2 — Lint, format, Vitest, path aliases

```markdown
## Summary
Add ESLint, Prettier (or project default), Vitest, Testing Library, `@/` path alias.

## Tasks
- [ ] lint + format scripts
- [ ] vitest config
- [ ] path alias `@` → `src`
- [ ] at least one smoke test

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-1
```

#### INIT-3 — libs + Atomic Design + feature folder skeleton

```markdown
## Summary
Ensure committed folder tree matches system design: `libs/{api,hooks,utils}`, Atomic `components/` (organisms for domain UI; no `src/features/`).

## Tasks
- [ ] Keep/align `frontend/src/libs/{api,hooks,utils,types}`
- [ ] Atomic `components/` (atoms → organisms → pages; no `features/`)
- [ ] No feature-level backend `api/` modules
- [ ] README stubs per libs package

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-1
```

#### INIT-4 — libs/api store + RTK Query baseApi

```markdown
## Summary
Wire configureStore with RTK Query baseApi under `src/libs/api`.

## Tasks
- [ ] `src/libs/api/baseApi.ts` with tagTypes
- [ ] `src/libs/api/store.ts`, `hooks.ts`
- [ ] Provider in app shell imports store from `@/libs/api`

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-1, INIT-2
```

#### INIT-5 — OpenAPI codegen pipeline

```markdown
## Summary
Generate API types (and optionally endpoints) from backend OpenAPI into libs.

## Tasks
- [ ] `npm run codegen:api` script
- [ ] Output under `src/libs/api/generated/`
- [ ] Document: never hand-edit generated files
- [ ] CI note or README check

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-4
```

#### INIT-6 — App shell, router, env, Markets placeholder

```markdown
## Summary
React Router + env config + placeholder Markets page route.

## Tasks
- [ ] `src/config/env.ts`, `constants.ts`
- [ ] routes: `/` or `/markets` → placeholder
- [ ] basic layout chrome (header title)

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-4
```

#### INIT-7 — Docs: frontend README + AGENTS.md + root links

```markdown
## Summary
Document how to run frontend with backend; nested AGENTS.md conventions.

## Tasks
- [ ] `frontend/README.md`
- [ ] `frontend/AGENTS.md`
- [ ] Update root README if needed
- [ ] Changelog Unreleased note if user-visible

## Labels
`frontend`, `type::chore`, `docs`, `priority::p0`

## Depends on
INIT-6, INIT-5
```

#### INIT-8 — Ant Design provider + theme

```markdown
## Summary
Install antd; ConfigProvider; theme tokens; optional dark algorithm.

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-1
```

#### INIT-9 — lightweight-charts + wrapper stub

```markdown
## Summary
Install lightweight-charts; chart host stub; candle mappers live in libs/utils later.

## Labels
`frontend`, `type::chore`, `priority::p0`

## Depends on
INIT-1, INIT-3
```

---

## Epic 2 — Multi-exchange spot markets (AFTER init)

| Field | Value |
|---|---|
| **Title** | `[frontend] Multi-exchange spot markets` |
| **Type** | Epic |
| **Labels** | `frontend`, `type::feature`, `priority::p1` |
| **Blocked by** | Epic 1 complete |
| **Design** | `docs/features/multi-exchange-spot-markets.md` |

### Epic description

```markdown
## Summary

Ship the first product UI feature: browse spot markets on Binance, Coinbase, and Bybit with search, filters, sort, pagination, and live refresh.

## Backend APIs (ready)

- `GET /api/v1/market/exchanges`
- `GET /api/v1/market/tags`
- `GET /api/v1/market/spot`

## Design

- `docs/features/multi-exchange-spot-markets.md`
- `docs/design/frontend-system-design.md`

## Blocked by

Epic: Frontend project initialization

## Child issues

MKT-1 … MKT-7
```

### Child issues — Epic 2

#### MKT-1 — RTK market endpoints (`libs/api`)

```markdown
## Summary
Inject RTK Query endpoints in `libs/api/endpoints/marketApi.ts` for exchanges, tags, and spot list.

## APIs
GET /api/v1/market/exchanges
GET /api/v1/market/tags
GET /api/v1/market/spot

## Labels
`frontend`, `type::feature`, `priority::p1`

## Depends on
Epic 1 done
```

#### MKT-2 — Markets page shell + ExchangeTabs

```markdown
## Summary
Markets page layout and exchange switcher (binance | coinbase | bybit).

## Labels
`frontend`, `type::feature`

## Depends on
MKT-1
```

#### MKT-3 — MarketsTable + formatters

```markdown
## Summary
Atomic MarketsTable organism: symbol, last, change%, volume, mcap, tags.

## Labels
`frontend`, `type::feature`

## Depends on
MKT-2
```

#### MKT-4 — Toolbar filters (search, quote, tags, sort)

```markdown
## Summary
Wire q, quote, tag, sort, order to spot query with debounce.

## Labels
`frontend`, `type::feature`

## Depends on
MKT-3
```

#### MKT-5 — Pagination + URL query sync

```markdown
## Summary
limit/offset controls; persist filters in URL search params.

## Labels
`frontend`, `type::feature`

## Depends on
MKT-4
```

#### MKT-6 — Live poll + visibility pause

```markdown
## Summary
Default 10s poll; pause when document hidden; refetch on focus.

## Labels
`frontend`, `type::feature`

## Depends on
MKT-3
```

#### MKT-7 — Empty/error UX + tests

```markdown
## Summary
Loading/empty/error/429 states; unit tests for helpers and table behaviors.

## Labels
`frontend`, `type::feature`, `test`

## Depends on
MKT-4, MKT-6
```

---

## Suggested milestone order

1. Create labels  
2. Create **Epic 1** + INIT issues; set epic relationship  
3. Implement INIT → merge to `develop`  
4. Create **Epic 2** + MKT issues  
5. Implement markets UI  

---

## API create script

See `docs/pm/create-gitlab-epics.sh` (requires `GITLAB_TOKEN` + `GITLAB_HOST`).

**Last updated:** 2026-07-26
