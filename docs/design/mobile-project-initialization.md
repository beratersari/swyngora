# Design: Mobile project initialization

**Status:** Implemented (web-first Chrome via react-native-web)

**Run:** `cd mobile && npm run web` → http://localhost:5180  
**Epic label:** `mobile` · `type::epic` · `epic::mobile-init`  
**Priority:** P0 — **must complete before multi-exchange spot markets mobile UI**  
**Branch pattern:** `feature/mobile-init-*` from `develop`  
**System design:** `docs/design/mobile-system-design.md`  
**Local PM:** `project-management/epics/mobile-project-initialization.md`  
**Stack decision:** `project-management/decisions/002-react-native-cli-modules-viewmodel.md`

---

## 1. Problem / goal

`mobile/` is reserved for the product React Native app but has **no toolchain**. Agents and humans need a **defined scaffold** before shipping mobile UI features.

**Goal:** Initialize a maintainable **React Native CLI + TypeScript** app (no Expo) that encodes Swyngora conventions:

| Convention | Mobile approach |
|---|---|
| UI structure | Atomic Design under `src/components/` — **no pages** |
| Features | `src/modules/<name>/` — **each module owns its pages** |
| Page pattern | **View + ViewModel** (`Page.tsx` + `Page.viewModel.ts`) |
| Data | RTK Query + OpenAPI codegen under `src/libs/api` |
| Colors | **Same brand tokens as frontend** (`frontend/src/styles/tokens/colors.ts`) |
| Background hygiene | AppState-aware polling (§6.6) |

So feature work can start with multi-exchange spot markets on mobile after init.

---

## 2. In scope

1. React Native **CLI** TypeScript project under `mobile/` (**no Expo**)
2. ESLint, Prettier, Jest + React Native Testing Library, path aliases (`@/` → `src/`)
3. Folder skeleton:
   - `src/components/{atoms,molecules,organisms,templates}` — Atomic only
   - `src/modules/{app,markets}/pages/` — module-owned pages
   - `src/libs/{api,hooks,utils,types}`
   - `src/app/` — App, providers, navigation
   - `src/styles/tokens/` — **copy frontend color (and related) tokens**
4. Redux Toolkit store + RTK Query `baseApi` under `src/libs/api/`
5. OpenAPI codegen into `src/libs/api/generated/`
6. App providers (Redux + Theme + SafeArea + Navigation)
7. React Navigation shell (tabs + stacks); routes `Home` / `MarketsList`
8. Theme: StyleSheet + **identical brand hex / semantic colors** to web
9. Core atoms: `Text`, `Button`, `Skeleton` + `ScreenTemplate`
10. Stub pages with ViewModels: Home (health) + Markets (placeholder)
11. Env: absolute `apiBaseUrl` in `src/config/env.ts` (Android debug cleartext)
12. Package `README.md` + nested `AGENTS.md`
13. Smoke scripts: `start`, `android`, `ios`, `test`, `lint`, `typecheck`, `codegen:api`
14. Local PM tasks under `project-management/` kept accurate

## 3. Out of scope

- Full markets table / coin detail UI (next mobile epic)
- Auth, watchlist, AI chat, paper trading
- Expo managed workflow / Expo Router
- Shared monorepo `packages/api-types` (optional later)
- App Store / Play release pipelines
- Chart library (deferred; DOM Lightweight Charts will not run on RN)
- i18n framework
- iOS green as hard DoD (Android required; iOS best-effort)

---

## 4. Technical choices (defaults)

| Choice | Default | Notes |
|---|---|---|
| Runtime | React Native CLI (community) | **No Expo** |
| Language | TypeScript strict | |
| Navigation | React Navigation (native stack + bottom tabs) | `gesture-handler` first import in `index.js` |
| State / server | Redux Toolkit + RTK Query | §6.9 |
| Codegen | `openapi-typescript` → `libs/api/generated` | Same OpenAPI as web |
| Package manager | npm | Lockfile committed |
| Unit tests | Jest + RNTL | preset `react-native` |
| Styling | StyleSheet + token files | No antd; no styled-components-RN in init |
| **Color tokens** | **Same as frontend** | See §5 |
| UI kit | Custom Atomic atoms first | React Native Paper only after a decision |
| Env | `src/config/env.ts` platform defaults | Absolute URL; no `react-native-config` in init |
| Acceptance device | Android first | WSL/Linux team |

---

## 5. Color tokens (locked to frontend)

**Source of truth (web):** `frontend/src/styles/tokens/colors.ts`  
**Mobile must ship an equivalent file** at `mobile/src/styles/tokens/colors.ts` with the **same hex values and semantic mapping**. Do not invent a second palette.

### Brand palette (copy verbatim)

| Token | Hex | Role |
|---|---|---|
| `navy` | `#111844` | App background, header, primary surface |
| `indigo` | `#4B5694` | Elevated surfaces, borders, secondary actions |
| `steel` | `#7288AE` | Muted text, icons, secondary labels |
| `cream` | `#EAE0CF` | Primary text on dark, accents, highlights |

### Semantic (same mapping as web)

Copy `semanticColors` structure from frontend:

- `bg.canvas` → navy  
- `bg.elevated` → indigo  
- `bg.muted` → `#1a2250`  
- `bg.inverse` → cream  
- `text.primary` → cream  
- `text.secondary` → steel  
- status / chart / skeleton keys as in web `colors.ts`

### Also mirror for layout consistency

| Token set | Web path | Mobile path |
|---|---|---|
| Colors | `frontend/src/styles/tokens/colors.ts` | `mobile/src/styles/tokens/colors.ts` |
| Spacing | `frontend/src/styles/tokens/spacing.ts` | `mobile/src/styles/tokens/spacing.ts` (same numeric grid) |
| Typography scale | `frontend/src/styles/tokens/typography.ts` | `mobile/src/styles/tokens/typography.ts` (sizes/weights; RN font stack as available) |

**Rule:** If frontend brand colors change, update mobile tokens in the **same MR** (or open a follow-up mobile task immediately). Prefer identical exports (`colors`, `semanticColors`) so agents do not drift.

---

## 6. Architecture snapshot

```text
mobile/src/
├── app/                 # App, providers, RootNavigator
├── components/          # Atomic ONLY (no pages/)
│   ├── atoms/
│   ├── molecules/
│   ├── organisms/
│   └── templates/
├── modules/             # Feature modules — own pages
│   ├── app/pages/HomePage/      # View + ViewModel
│   └── markets/pages/MarketsPage/
├── libs/api/            # RTK Query + OpenAPI generated
├── libs/hooks/          # useAppStateActive, …
├── config/env.ts        # absolute API base URL
└── styles/tokens/       # same colors as frontend
```

**Hard rules**

1. `components/**` never imports `modules/**`
2. `libs/**` never imports modules, app, or components
3. Pages do not call RTK directly — ViewModels do
4. ViewModels return plain data + callbacks (no JSX)
5. No Expo packages

Full detail: `docs/design/mobile-system-design.md`.

---

## 7. Deliverables checklist

### 7.1 Toolchain

- [ ] `package.json` scripts: `start`, `android`, `ios`, `test`, `lint`, `typecheck`, `codegen:api`
- [ ] TypeScript strict + path alias `@/*` → `src/*`
- [ ] ESLint (incl. `no-restricted-imports` boundaries) + Prettier
- [ ] Jest + RNTL smoke setup
- [ ] `.env.example` documents API base URL (docs only; runtime via `env.ts`)
- [ ] Android **debug** cleartext for local HTTP backend
- [ ] **No** `expo` dependency in package.json

### 7.2 Source bootstrap

- [ ] `index.js` — first import: `react-native-gesture-handler`; AppRegistry → `src/app/App`
- [ ] `src/app/providers.tsx` — Redux, Theme, SafeAreaProvider, NavigationContainer
- [ ] `src/app/navigation/RootNavigator.tsx` — Home + Markets tabs/stacks
- [ ] `src/libs/api/baseApi.ts`, `store.ts`, `hooks.ts`
- [ ] `src/libs/api/endpoints/healthApi.ts` — `useGetHealthQuery`
- [ ] `src/config/env.ts` — absolute `apiBaseUrl` (Android emulator `10.0.2.2:8080` default)
- [ ] `src/styles/tokens/colors.ts` — **matches frontend hex**
- [ ] Atoms Text / Button / Skeleton + ScreenTemplate
- [ ] `modules/app/pages/HomePage` — View + ViewModel (health)
- [ ] `modules/markets/pages/MarketsPage` — stub View + ViewModel

### 7.3 Codegen

- [ ] Script generates into `src/libs/api/generated/`
- [ ] README: run after OpenAPI changes; never hand-edit; **also refresh frontend** when OpenAPI changes
- [ ] Smoke import of generated type

### 7.4 Conventions docs

- [ ] `mobile/README.md` — toolchain pins, Android run, env, cleartext, scripts
- [ ] `mobile/AGENTS.md` — Atomic, modules, ViewModel, no Expo, token parity, boundaries
- [ ] Root README / AGENTS note for `mobile/` layout exception (pages in modules)

### 7.5 Quality gates

- [ ] `npm test` passes
- [ ] `npm run typecheck` / `lint` pass
- [ ] `npm run android` launches shell (or documented host blocker)
- [ ] Home health succeeds vs local backend when server is up

---

## 8. Acceptance criteria

1. Fresh clone: from `mobile/`, install deps and `npm run android` shows shell (documented host).
2. App navigates **Home** and **MarketsList** stubs.
3. Home ViewModel uses `useGetHealthQuery` with AppState-aware polling; View has no RTK imports.
4. `npm run codegen:api` succeeds against current OpenAPI.
5. `src/styles/tokens/colors.ts` brand hex equals `frontend/src/styles/tokens/colors.ts`.
6. No `components/pages/` directory; no Expo deps.
7. ESLint forbids components → modules and libs → modules/components imports.
8. Structure matches `mobile/AGENTS.md` and system design.

---

## 9. Implementation notes

### baseApi sketch

```ts
// src/libs/api/baseApi.ts
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { env } from '@/config/env';

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({ baseUrl: env.apiBaseUrl }),
  tagTypes: ['SpotList', 'Exchange', 'ProductTag', 'Watchlist', 'Health'],
  endpoints: () => ({}),
});
```

### env sketch (absolute URL required)

```ts
// src/config/env.ts
import { Platform } from 'react-native';

/** Optional one-line override for physical devices / custom hosts. */
const DEV_OVERRIDE: string | null = null;

const defaults = {
  android: 'http://10.0.2.2:8080',
  ios: 'http://localhost:8080',
  default: 'http://localhost:8080',
} as const;

function resolveApiBaseUrl(): string {
  if (DEV_OVERRIDE) return DEV_OVERRIDE;
  if (Platform.OS === 'android') return defaults.android;
  if (Platform.OS === 'ios') return defaults.ios;
  return defaults.default;
}

export const env = {
  apiBaseUrl: resolveApiBaseUrl(),
} as const;
```

### ViewModel testing

Prefer injectable `viewModel` prop on the page for View tests; mock RTK hooks in ViewModel unit tests.

### Dependency on backend

Init unit tests do **not** require a running backend.  
Manual DoD: start `go run ./cmd/server` from `backend/` and confirm Home health on Android emulator.

---

## 10. Issue breakdown (local PM)

| ID | Title | Estimate |
|---|---|---|
| MINIT-1 | Scaffold React Native CLI TypeScript app (no Expo) | M |
| MINIT-2 | Lint, format, Jest, path aliases | S |
| MINIT-3 | libs + Atomic + modules folder skeleton + boundary ESLint | M |
| MINIT-4 | libs/api store + RTK Query baseApi + env | M |
| MINIT-5 | OpenAPI codegen → libs/api/generated | M |
| MINIT-6 | Navigation shell + AppState hook + providers | M |
| MINIT-7 | Color tokens (match frontend) + Text/Button/Skeleton + ScreenTemplate | M |
| MINIT-8 | Home + Markets stub pages with ViewModels | M |
| MINIT-9 | Package README + AGENTS.md + root links + changelog | S |

**Recommended MR grouping:**

| MR | Tasks |
|---|---|
| MR1 | MINIT-1 + MINIT-2 + MINIT-3 |
| MR2 | MINIT-4 + MINIT-5 |
| MR3 | MINIT-6 + MINIT-7 |
| MR4 | MINIT-8 + MINIT-9 |

Or follow the finer PR plan in `docs/design/mobile-system-design.md` §PR Plan.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| Expo creep by agents | `mobile/AGENTS.md` + reject any `expo` dep in review |
| Color drift from web | Explicit token parity in DoD; same file shape as frontend |
| WSL cannot run emulator | Document Android host (Windows Android Studio / physical device); cleartext config |
| Pages under components | ESLint boundaries + no `components/pages` folder |
| Fat screens | Mandatory ViewModel per page from first stub |
| Dual codegen drift | Document: OpenAPI change → regenerate **frontend + mobile** |

---

## 12. Definition of done

- [ ] All acceptance criteria met  
- [ ] Changelog Unreleased: “Add product mobile (React Native) scaffold”  
- [ ] Epic closed only when Android shell runs, codegen works, tokens match frontend  
- [ ] Unblocks next epic: multi-exchange spot markets (mobile)  

**Last updated:** 2026-07-26
