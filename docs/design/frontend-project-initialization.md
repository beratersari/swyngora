# Design: Frontend project initialization

**Status:** Ready for implementation  
**Epic label:** `frontend` · `type::epic` · `epic::frontend-init`  
**Priority:** P0 — **must complete before multi-exchange spot markets UI**  
**Branch pattern:** `feature/frontend-init-*` from `develop`  
**Docs home:** `docs/design/frontend-system-design.md`

---

## 1. Problem / goal

`frontend/` is reserved for the product React app but has no toolchain. Agents and humans need a **defined scaffold** before shipping UI features.

**Goal:** Initialize a maintainable React + TypeScript app that encodes Swyngora conventions (Atomic Design, **Ant Design**, **Lightweight Charts**, RTK Query, OpenAPI codegen, `libs/`) so feature work can start with multi-exchange spot markets.

---

## 2. In scope

1. Vite + React 18+ + TypeScript project under `frontend/`
2. ESLint, Prettier (or biome if chosen), Vitest + Testing Library
3. Path aliases (`@/` → `src/`)
4. Folder skeleton with **`src/libs/{api,hooks,utils,types}`** + Atomic `components/` + `features/markets`
5. Redux Toolkit store + RTK Query `baseApi` under **`src/libs/api/`**
6. OpenAPI codegen into **`src/libs/api/generated/`**
7. App providers (Redux + **Ant Design ConfigProvider**), React Router, empty Markets route placeholder
8. Install **antd** (+ icons as needed) and theme baseline
9. Install **lightweight-charts** + thin chart host stub
10. Env sample: `VITE_API_BASE_URL`
11. Package `README.md` + nested `AGENTS.md` (libs + antd + charts rules)
12. Smoke scripts: `dev`, `build`, `test`, `lint`, `codegen:api`
13. Local PM tasks under `project-management/` kept accurate

## 3. Out of scope

- Full markets table UI (Epic B)
- Auth, watchlist, charts
- Deploy pipeline / Docker image
- Shared monorepo package for types (optional later)

---

## 4. Technical choices (defaults)

| Choice | Default | Notes |
|---|---|---|
| Bundler | Vite | SPA |
| Language | TypeScript strict | |
| UI library | React | |
| UI kit | **Ant Design (`antd`)** | ConfigProvider; Atomic wrappers preferred |
| Charts | **lightweight-charts** | Stub in init; full candles in detail epic |
| Router | React Router v6+ | |
| State / server | Redux Toolkit + RTK Query | §6.9 |
| Codegen | `openapi-typescript` + hand `injectEndpoints` **or** `@rtk-query/codegen-openapi` | Document command |
| Package manager | npm | Lockfile committed |
| Unit tests | Vitest | |
| Styling | Ant Design theme tokens + app CSS | No second UI kit |

---

## 5. Deliverables checklist

### 5.1 Toolchain

- [ ] `package.json` with scripts: `dev`, `build`, `preview`, `test`, `lint`, `codegen:api`
- [ ] `tsconfig.json` (strict), Vite React plugin
- [ ] `.env.example` with `VITE_API_BASE_URL=http://localhost:8080`
- [ ] `.gitignore` entries for `node_modules`, `dist`, `.env`

### 5.2 Source bootstrap

- [ ] `index.html` + `src/main.tsx`
- [ ] `src/app/providers.tsx` — Redux Provider, Ant Design `ConfigProvider`, Router
- [ ] `src/app/routes.tsx` — `/` → Markets placeholder page
- [ ] `src/libs/api/baseApi.ts` — `createApi` with `fetchBaseQuery`
- [ ] `src/libs/api/store.ts` + `src/libs/api/hooks.ts` (typed Redux hooks)
- [ ] `src/libs/api/endpoints/` ready for injectEndpoints
- [ ] `src/libs/hooks/`, `src/libs/utils/` packages with README + barrel
- [ ] `src/config/env.ts` + `constants.ts`

### 5.3 Codegen

- [ ] Script that generates into `src/libs/api/generated/`
- [ ] README documents: run after OpenAPI changes; never hand-edit generated files
- [ ] At least one smoke import of a generated type in baseApi or a stub endpoint

### 5.4 Conventions docs

- [ ] `frontend/README.md` — run backend + frontend, env, scripts
- [ ] `frontend/AGENTS.md` — Atomic rules, codegen, no fetch in atoms
- [ ] Root README link still accurate

### 5.5 Quality gates

- [ ] `npm test` passes (at least one smoke test)
- [ ] `npm run build` succeeds
- [ ] `npm run lint` clean (or documented baseline)

---

## 6. Acceptance criteria

1. Fresh clone: from `frontend/`, `npm install && npm run dev` serves the app.
2. App shows a **placeholder Markets** route (title only is OK).
3. Changing `VITE_API_BASE_URL` is documented and read once at boot.
4. `npm run codegen:api` runs without error against current OpenAPI.
5. Directory layout matches system design §4.
6. No production business UI required beyond shell.

---

## 7. Implementation notes

### baseApi sketch

```ts
// src/libs/api/baseApi.ts
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { env } from '@/config/env';

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({ baseUrl: env.apiBaseUrl }),
  tagTypes: ['SpotList', 'Exchange', 'ProductTag', 'Watchlist'],
  endpoints: () => ({}),
});
```

### Suggested first smoke test

```ts
// src/config/env.test.ts
import { describe, it, expect } from 'vitest';
import { env } from './env';

describe('env', () => {
  it('exposes apiBaseUrl string', () => {
    expect(typeof env.apiBaseUrl).toBe('string');
    expect(env.apiBaseUrl.length).toBeGreaterThan(0);
  });
});
```

### Dependency on backend

Init does **not** require a running backend for unit tests.  
Dev UX: document starting `go run ./cmd/server` for later feature work.

---

## 8. Issue breakdown (GitLab)

| IID (local) | Title | Estimate |
|---|---|---|
| INIT-1 | Scaffold Vite React TypeScript app | S |
| INIT-2 | Add lint, format, Vitest, path aliases | S |
| INIT-3 | Commit libs + Atomic Design + feature folder skeleton | S |
| INIT-4 | Wire `libs/api` store + RTK Query baseApi | M |
| INIT-5 | OpenAPI codegen pipeline → `libs/api/generated` | M |
| INIT-6 | App shell, router, env, Markets placeholder (Ant Layout) | M |
| INIT-7 | Package README + AGENTS.md + root links | S |
| INIT-8 | Ant Design provider + theme | M |
| INIT-9 | lightweight-charts + chart wrapper stub | S |

All issues blocked by nothing except sequential merge preference: INIT-1 → INIT-2 → … or squash into 1–2 MRs if small team.

**Recommended MR grouping:**

- MR1: INIT-1 + INIT-2 + INIT-3  
- MR2: INIT-4 + INIT-5  
- MR3: INIT-6 + INIT-7  

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| Over-scaffolding unused libs | Only add deps needed for RTK + router + test |
| Codegen tool churn | Document exact package versions in README |
| Agents inventing structure | Nested `AGENTS.md` is source of truth |

---

## 10. Definition of done

- [ ] All acceptance criteria met  
- [ ] Changelog draft note under Unreleased if user-visible (“Add product frontend scaffold”)  
- [ ] Epic closed only when placeholder app runs and codegen works  
- [ ] Unblocks Epic B: Multi-exchange spot markets  

**Last updated:** 2026-07-26
