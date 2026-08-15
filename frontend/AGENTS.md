# AGENTS.md — frontend/

Package conventions for the **product web UI**. Closest `AGENTS.md` wins under this tree; user chat overrides docs. Root `AGENTS.md` still applies for Git Flow, SemVer, and monorepo rules.

---

## 1. Role

React SPA that talks to the Go backend via **OpenAPI-described HTTP**. Not `simple-frontend/`.

## 2. Stack (mandatory)

| Concern            | Choice                                                                                                                          |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| UI kit             | **Ant Design** (`antd`) — wrap in Atomic components when practical                                                              |
| Charts             | **TradingView Lightweight Charts** (`lightweight-charts`)                                                                       |
| UI structure       | Atomic Design (atoms → molecules → organisms → templates → pages)                                                               |
| File split (UI)    | **Prefixed** colocation: `Name.tsx`, `Name.types.ts`, `Name.styles.ts`, `Name.constants.ts`, `Name.helpers.ts`, `Name.test.tsx` |
| Shared non-UI      | **`src/libs/{api,realtime,hooks,utils,types}`**                                                                                 |
| Server state       | **RTK Query** only for backend REST — lives in **`libs/api`**                                                                   |
| Types from backend | **Generated** into `libs/api/generated/` from OpenAPI                                                                           |
| Bundle             | Vite + React + TypeScript                                                                                                       |
| Design system      | Tokens + Text + Skeleton + motion — `docs/design/frontend-design-system.md`                                                     |
| Styling            | **styled-components only** — colocate `*.styles.ts` (no CSS/CSS modules)                                                        |
| Brand colors       | Light CoinMarketCap-like: paper `#FFFFFF` / blue `#3861FB` / up `#16C784` / down `#EA3943` — see `styles/tokens/colors.ts` |
| Loading            | All content components support `isLoading` → Skeleton                                                                           |
| Localization       | **i18next** + **react-i18next** under `src/libs/i18n/` — locale JSON catalogs; no hard-coded UI copy                          |

**Decision record:** `project-management/decisions/001-antd-and-lightweight-charts.md`  
**Design system:** `docs/design/frontend-design-system.md`

## 3. Folder map (Option A — no `features/`)

| Path | Role |
| --- | --- |
| `src/components/atoms` | Design-system primitives |
| `src/components/molecules` | Small compositions / chart hosts |
| `src/components/organisms` | Domain UI sections (markets table, detail panels) — **props only** |
| `src/components/templates` | Layout shells without data |
| `src/components/pages` | Route screens — **RTK Query only here** |
| `src/libs/api/` | baseApi, store, endpoints, OpenAPI generated |
| `src/libs/hooks/` | Shared React hooks |
| `src/libs/utils/` | Pure helpers (incl. candle → chart mappers) |
| `src/libs/types/` | Shared view/re-export types |
| `src/app/` | Providers (Redux, Ant `ConfigProvider`), router |
| `src/config/` | Env + app constants |
| `src/libs/i18n/` | i18n init, locale JSON (`en`, `tr`), Ant Design locale bridge |

**Do not use `src/features/`** for product UI. Domain widgets live under **organisms**; screens under **pages**. Revisit feature folders only when multiple product areas need isolation (e.g. markets + watchlist + paper + AI).

**UI strings:** use `useTranslation` / `t('namespace:key')` — catalogs in `libs/i18n/locales/`. Exchange ids and symbols stay untranslated.

Full design: `docs/design/frontend-system-design.md`.  
Local tasks: `project-management/`.

## File naming (searchable colocation)

Every component module uses the **component name as a prefix** so search/filter finds related files easily:

```text
components/atoms/Text/
├── Text.tsx              # component
├── Text.types.ts         # props / local types
├── Text.styles.ts        # styled-components
├── Text.constants.ts     # local constants (not bare constants.ts)
├── Text.helpers.ts       # pure helpers (not bare helpers.ts)
├── Text.test.tsx         # tests
└── index.ts              # public barrel
```

| Suffix              | Purpose                |
| ------------------- | ---------------------- |
| `Name.tsx`          | Component              |
| `Name.types.ts`     | Props, view models     |
| `Name.styles.ts`    | styled-components      |
| `Name.constants.ts` | Magic values, defaults |
| `Name.helpers.ts`   | Pure functions         |
| `Name.test.tsx`     | Unit tests             |

**Do not** use bare `constants.ts` / `helpers.ts` / `styles.ts` inside component folders — always prefix with the component name.

App-level config may use `src/config/constants.ts` (not a component).

## 4. Ant Design rules

1. App-wide theme via `ConfigProvider` in `app/providers.tsx` (INIT-8).
2. Prefer **Atomic wrappers** (`components/atoms/Button`, etc.) over raw `antd` deep in the tree — **organisms** may use antd Table/Form when wrappers do not exist yet; promote wrappers as patterns stabilize.
3. Use Ant Design **Table, Tabs, Form, Select, Input, Layout, Typography, Tag, Pagination, Spin, Alert** for markets UI.
4. Do not mix a second full UI kit (MUI, Chakra, etc.) without a decision doc.

## 5. Chart rules (Lightweight Charts)

1. Dependency: `lightweight-charts` (INIT-9).
2. Mount charts in a dedicated molecule/organism (e.g. candle chart host); do not put chart lifecycle in atoms that only render icons/text.
3. Map API candle **strings** → chart numbers in **`libs/utils`**.
4. Primary use: OHLCV from `GET /api/v1/market/candles` on detail views (later epic). Markets list phase uses tables, not charts.
5. Do not add ECharts/Recharts/Chart.js as a second default without a new decision.

## 6. Hard rules

1. **API layer only under `libs/api`** — never under components or a `features/*/api` tree.
2. **Shared hooks under `libs/hooks`**.
3. **Shared pure code under `libs/utils`**.
4. **Do not** call `fetch`/`axios` / RTK from atoms, molecules, or organisms — **pages only**.
5. **Do not** hand-edit `libs/api/generated/`.
6. **Do not** hand-maintain long-term API DTOs that duplicate OpenAPI.
7. **libs must not import** pages or organisms.
8. No upward Atomic imports (atoms must not import organisms/pages).
9. **No `src/features/`** unless the team explicitly revisits Option B (feature-owned pages).
10. User-visible work lands with tests and docs.
11. Keep this file and `README.md` accurate after structural/stack changes.

## 7. Preferred imports

```ts
import { useListSpotMarketsQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { formatPrice } from '@/libs/utils';
import { Button } from '@/components/atoms/Button';
import { MarketsTable } from '@/components/organisms/MarketsTable';
```

## 8. First epics

| Order | Epic                        | Local PM                                                      |
| ----- | --------------------------- | ------------------------------------------------------------- |
| 1     | Project initialization      | `project-management/epics/frontend-project-initialization.md` |
| 2     | Multi-exchange spot markets | `project-management/epics/multi-exchange-spot-markets.md`     |

Do not implement Epic 2 UI before Epic 1 acceptance criteria pass.

## 9. Local run (post-init)

```bash
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

## 10. Codegen

```bash
npm run codegen:api
# output: src/libs/api/generated/
```

## 11. Format / lint / test

```bash
npm run format         # Prettier write
npm run format:check   # CI check
npm run lint           # ESLint (+ eslint-config-prettier)
npm test               # Vitest
npm run test:coverage  # Vitest + @vitest/coverage-v8 (line + branch)
```

Config: `.prettierrc.json` · ignore: `.prettierignore`  
Do not format `src/libs/api/generated/`.

**Env (optional):** `VITE_API_BASE_URL`, `VITE_CLIENT_ID` (watchlist `X-Client-Id`).

**Vite on WSL `/mnt/c`:** `server.watch.usePolling` is on so HMR sees Windows-FS writes. Restart `npm run dev` if the browser still serves a stale module.

**Last updated:** 2026-08-11
