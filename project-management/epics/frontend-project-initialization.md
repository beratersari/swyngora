# Epic A: Frontend project initialization

**Priority:** P0  
**Status:** done  
**Blocks:** multi-exchange spot markets  
**Design:** `docs/design/frontend-project-initialization.md`

## Goal

Scaffold `frontend/` with Vite, React, TS, **Ant Design**, **Lightweight Charts** wrapper hook, `libs/{api,hooks,utils}`, RTK Query, OpenAPI codegen, app shell.

## Tasks

- [x] INIT-1 Scaffold Vite React TypeScript
- [x] INIT-2 Lint, format, Vitest, aliases
- [x] INIT-3 libs + Atomic folder skeleton
- [x] INIT-4 libs/api store + baseApi
- [x] INIT-5 OpenAPI codegen
- [x] INIT-6 App shell + router + Markets placeholder
- [x] INIT-7 Package docs
- [x] INIT-8 Ant Design provider + theme
- [x] INIT-9 lightweight-charts dependency + chart wrapper stub

## Acceptance

- `npm run dev` works with Ant Design Layout shell
- `npm run codegen:api` works
- Chart package installed; wrapper exports a placeholder or empty chart host
- Structure matches `frontend/AGENTS.md`
