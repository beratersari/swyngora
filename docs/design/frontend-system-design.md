# Frontend system design (product web)

**Status:** Draft for implementation  
**Audience:** Frontend engineers, agents  
**Scope:** Production web app under `frontend/`  
**Related:** root `AGENTS.md` §6.8 / §6.9, `backend/api/openapi/openapi.yaml`, `docs/features/market-data.md`  
**First milestone:** [Project initialization](./frontend-project-initialization.md)  
**First product feature:** [Multi-exchange spot markets](../features/multi-exchange-spot-markets.md)

---

## 1. Goals

Build a production React web client that:

1. Consumes the existing Swyngora Go HTTP API (OpenAPI-first).
2. Follows **Atomic Design** and **mandatory file separation** (`*.types.ts`, `constants.ts`, `helpers.ts` inside component folders).
3. Centralizes non-UI code under **`src/libs/`**: **api**, **hooks**, **utils** (and shared **types**).
4. Uses **RTK Query** for all backend HTTP, with **generated types/endpoints** from OpenAPI under `libs/api`.
5. Uses **Ant Design** for UI chrome and **TradingView Lightweight Charts** for OHLCV.
6. Starts with **multi-exchange spot markets** after project initialization.
7. Stays separate from `simple-frontend/` (test harness only).
8. Tracks work locally in **`project-management/`** until GitLab issues are mirrored.

Non-goals for the first two epics: auth, AI UI, paper trading/alerts, mobile.

---

## 2. Context

| Layer | Today |
|---|---|
| Backend | Multi-exchange spot (Binance, Coinbase, Bybit), candles, ticker, supply, indicators, client-id watchlist |
| Contract | `backend/api/openapi/openapi.yaml` |
| Test UI | `simple-frontend/` — reference behavior only |
| Product UI | `frontend/` — scaffolded; libs-first layout |

Primary list API:

```http
GET /api/v1/market/spot?exchange=binance&quote=USDT&sort=quoteVolume&limit=50&offset=0
```

Supporting: `/exchanges`, `/tags`, `/intervals`.

---

## 3. High-level architecture

```text
┌─────────────────────────────────────────────────────────────┐
│  app/  +  components/pages  +  features/*                   │
│  route entry, composition, feature UI                       │
└───────────────────────────┬─────────────────────────────────┘
                            │ import
┌───────────────────────────▼─────────────────────────────────┐
│  components/  (Atomic Design UI only)                       │
│  atoms → molecules → organisms → templates → pages          │
│  presentational; no RTK inside atoms/molecules              │
└───────────────────────────┬─────────────────────────────────┘
                            │ import
┌───────────────────────────▼─────────────────────────────────┐
│  libs/                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ libs/api    │  │ libs/hooks  │  │ libs/utils  │          │
│  │ RTK Query   │  │ shared      │  │ pure        │          │
│  │ baseApi     │  │ React hooks │  │ formatters  │          │
│  │ endpoints   │  │             │  │ builders    │          │
│  │ generated   │  │             │  │             │          │
│  │ store       │  │             │  │             │          │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘          │
│         │                │                                   │
│         └────── HTTP ────┘                                   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Swyngora backend :8080  (OpenAPI contract)                 │
└─────────────────────────────────────────────────────────────┘
```

**Dependency direction**

```text
app / features / components/pages
        →  libs/hooks  →  libs/api  →  backend
        →  libs/utils
components (atoms/molecules)  →  libs/utils only (optional)
```

- **libs must not import** `features/*` or page components.
- **features** import `libs/*` for data and shared behavior; keep feature-local UI under `features/<name>/components/`.

---

## 4. Target folder structure

```text
frontend/
├── README.md
├── AGENTS.md
├── package.json                 # INIT epic
├── vite.config.ts
├── tsconfig*.json
├── index.html
├── public/
├── scripts/
│   └── codegen-api.sh           # OpenAPI → libs/api/generated
└── src/
    ├── main.tsx
    ├── app/
    │   ├── App.tsx
    │   ├── routes.tsx
    │   └── providers.tsx        # Redux + Ant Design ConfigProvider
    ├── config/
    │   ├── constants.ts
    │   └── env.ts
    ├── components/              # Shared Atomic Design UI
    │   ├── atoms/
    │   ├── molecules/
    │   ├── organisms/
    │   ├── templates/
    │   └── pages/
    ├── features/
    │   └── markets/             # Multi-exchange spot UI (Epic B)
    │       ├── components/      # Feature UI (Atomic-split files)
    │       ├── constants.ts
    │       ├── markets.types.ts
    │       └── helpers.ts       # feature-only view mappers (optional)
    ├── libs/                    # ★ shared layers (api / hooks / utils)
    │   ├── api/
    │   │   ├── baseApi.ts
    │   │   ├── store.ts         # configureStore + api middleware
    │   │   ├── hooks.ts         # useAppDispatch / useAppSelector
    │   │   ├── endpoints/       # marketApi.ts, watchlistApi.ts, …
    │   │   ├── generated/       # OpenAPI output — DO NOT hand-edit
    │   │   └── index.ts
    │   ├── hooks/               # useDocumentVisible, useDebouncedValue, …
    │   ├── utils/               # formatPrice, exchange symbols, …
    │   └── types/               # shared view/re-export types (not DTOs)
    └── styles/
```

### 4.1 What lives in `libs/` vs feature folders

| Concern | Location |
|---|---|
| `createApi` / `baseQuery` / tag types | `libs/api/baseApi.ts` |
| OpenAPI codegen output | `libs/api/generated/` |
| Domain endpoints (`injectEndpoints`) | `libs/api/endpoints/*.ts` |
| Redux `configureStore` | `libs/api/store.ts` |
| Typed store hooks | `libs/api/hooks.ts` |
| Shared React hooks | `libs/hooks/` |
| Pure formatters / query builders | `libs/utils/` |
| Feature table UI, filters chrome | `features/markets/components/` |
| Page route composition | `components/pages/MarketsPage` (or feature export) |
| Component-local pure helpers | component folder `helpers.ts` (Atomic rule) |

**Do not** put backend REST modules under `features/*/api` — extend `libs/api/endpoints/` instead.

### 4.2 Component folder rule (unchanged Atomic split)

```text
components/organisms/MarketsTable/
├── MarketsTable.tsx
├── MarketsTable.types.ts
├── MarketsTable.styles.ts
├── MarketsTable.constants.ts
├── MarketsTable.helpers.ts
├── MarketsTable.test.tsx
└── index.ts
```

---

## 5. Data layer (`libs/api`)

### 5.1 OpenAPI is the contract

| Step | Owner |
|---|---|
| Change public HTTP | Backend updates `backend/api/openapi/openapi.yaml` |
| Regenerate client | `frontend`: `npm run codegen:api` → `src/libs/api/generated/` |
| Use in UI | Import RTK hooks from `@/libs/api` |

### 5.2 RTK Query defaults

| Concern | Default |
|---|---|
| Base URL | `VITE_API_BASE_URL` → `http://localhost:8080` |
| Spot poll | 10s UI; backend spot TTL ~5s |
| Pause poll | `libs/hooks` visibility helper + RTK `skip` / polling off |
| Refetch on focus | Yes |
| Cache tags | `SpotList`, `Exchange`, `ProductTag`, `Watchlist`, … |

### 5.3 Phase-1 endpoints (markets)

| Operation | Method | Path | Module |
|---|---|---|---|
| List exchanges | GET | `/api/v1/market/exchanges` | `libs/api/endpoints/marketApi.ts` |
| List tags | GET | `/api/v1/market/tags` | same |
| List spot | GET | `/api/v1/market/spot` | same |

### 5.4 baseApi sketch

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

```ts
// src/libs/api/endpoints/marketApi.ts
import { baseApi } from '../baseApi';

export const marketApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listExchanges: build.query(/* … */),
    listProductTags: build.query(/* … */),
    listSpotMarkets: build.query(/* … */),
  }),
});

export const {
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
} = marketApi;
```

---

## 6. UI information architecture (first screens)

| Route | Page | libs/api |
|---|---|---|
| `/` or `/markets` | Markets | exchanges, tags, spot |
| `/markets/:exchange/:symbol` | Detail (later) | ticker, candles, supply, indicators |
| `/watchlist` | Watchlist (later) | watchlist + spot |

---

## 7. Cross-cutting concerns

### Environment

| Variable | Purpose |
|---|---|
| `VITE_API_BASE_URL` | Backend origin |

### Freshness (AGENTS.md §6.6)

- Poll only while tab visible (`libs/hooks`).
- Handle 502 mcap / 429 rate limit in page-level UI state.

### Exchange quirks

Binance/Bybit `BTCUSDT` vs Coinbase `BTC-USD` — helpers in `libs/utils/exchange.ts`.

### Testing

| Layer | Place |
|---|---|
| utils | `libs/utils/*.test.ts` |
| hooks | `libs/hooks/*.test.ts` |
| endpoints | `libs/api` with mocked baseQuery |
| UI | component colocated tests |

### Path aliases (init epic)

```ts
// tsconfig / vite
"@/*" → "src/*"
// preferred imports:
import { useListSpotMarketsQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { formatPrice } from '@/libs/utils';
```

---

## 8. Key decisions

| Decision | Choice | Rationale |
|---|---|---|
| Non-UI shared code | **`src/libs/{api,hooks,utils}`** | Clear layers; features stay UI-focused |
| API location | All RTK under **`libs/api`** | Single data layer; no feature-level fetch modules |
| App location | `frontend/` | Product app |
| Bundler | Vite + React + TS | DX |
| UI kit | **Ant Design (`antd`)** | Tables, forms, layout, theme for trading dashboard |
| Charts | **TradingView Lightweight Charts** | Financial OHLCV candlesticks; free Apache-2.0 |
| UI structure | Atomic Design wrapping antd | AGENTS.md §6.8 |
| Data | RTK Query + OpenAPI codegen | AGENTS.md §6.9 |
| First feature | Multi-exchange spot | Stable backend APIs |
| Local PM | **`project-management/`** | Tasks/epics until GitLab MCP fully used |
| Styling | Ant Design theme tokens (+ app CSS as needed) | ConfigProvider; avoid second UI kit |

---

## 9. Delivery plan (epics)

### Epic A — Frontend project initialization (**first**)

Scaffold toolchain, **libs** skeleton, store in `libs/api`, codegen → `libs/api/generated`, empty shell.  
See [frontend-project-initialization.md](./frontend-project-initialization.md).

### Epic B — Multi-exchange spot markets

Markets UI consuming `@/libs/api` market endpoints.  
See [multi-exchange-spot-markets.md](../features/multi-exchange-spot-markets.md).

---

## 10. PR plan

| Order | PR | Depends on |
|---|---|---|
| 1 | Scaffold Vite/React/TS + lint/test | — |
| 2 | `libs/{api,hooks,utils}` + Atomic `components/` tree | 1 |
| 3 | Ant Design ConfigProvider + theme (INIT-8) | 2 |
| 4 | lightweight-charts dep + chart host stub (INIT-9) | 2 |
| 5 | `libs/api` baseApi + store + OpenAPI codegen | 2 |
| 6 | App shell + routing + env (Ant Layout) | 3, 5 |
| 7 | Markets endpoints in `libs/api/endpoints` + Markets page (Ant Table) | 6 |
| 8 | Filters, tags, sort, pagination | 7 |
| 9 | Live poll + visibility (`libs/hooks`) | 7 |
| 10 | Polish + tests | 8, 9 |

PRs 1–6 = Epic A. PRs 7–10 = Epic B.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| Feature reintroduces local `api/` | Code review + AGENTS.md hard rule |
| OpenAPI drift | Codegen in same MR as backend contract |
| Circular imports libs ↔ features | libs never import features |

---

## 12. Open questions

1. Persist last exchange/quote in `localStorage`? (Default: yes + URL params.)  
2. npm vs pnpm? (Default: npm.)  
3. Ant Design dark algorithm as default for markets? (Default: yes for trading feel.)

---

## 13. Success criteria

- [ ] `frontend/` runs with `npm install && npm run dev`  
- [ ] Types/hooks generate into `libs/api/generated`  
- [ ] Shared data access only via `@/libs/api`  
- [ ] Multi-exchange spot markets browseable after Epic B  
- [ ] Structure documented in package AGENTS.md  

**Last updated:** 2026-07-26 (antd + lightweight-charts)
