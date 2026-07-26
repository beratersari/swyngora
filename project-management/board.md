# Product client board

Local status for web (`frontend/`) and mobile (`mobile/`) until GitLab issues are fully wired.

## Frontend stack (locked)

- UI: **Ant Design**
- Charts: **TradingView Lightweight Charts**
- Data: RTK Query + OpenAPI in `libs/api`
- Colors: `frontend/src/styles/tokens/colors.ts`

## Mobile stack (locked)

- Runtime: **React Native CLI** (**no Expo**)
- UI: Atomic atoms + StyleSheet (no antd)
- Architecture: **`modules/*/pages`** + **ViewModel**; components never hold pages
- Data: RTK Query + OpenAPI in `libs/api`
- **Colors: same tokens as frontend** (`navy` `#111844`, `indigo` `#4B5694`, `steel` `#7288AE`, `cream` `#EAE0CF`)
- Decision: `decisions/002-react-native-cli-modules-viewmodel.md`

---

## Epic A — Frontend project initialization (P0, done)

| ID | Task | Status |
|---|---|---|
| INIT-1 | Scaffold Vite React TypeScript app | done |
| INIT-2 | Lint, format, Vitest, path aliases | done |
| INIT-3 | libs + Atomic folders + Ant Design provider | done |
| INIT-4 | libs/api store + RTK baseApi | done |
| INIT-5 | OpenAPI codegen → libs/api/generated | done |
| INIT-6 | App shell, router, env, Markets placeholder (Ant Layout) | done |
| INIT-7 | Docs: README + AGENTS + changelog | done |
| INIT-8 | Install/configure Ant Design + theme tokens | done |
| INIT-9 | Add lightweight-charts + thin chart wrapper atom/molecule | done |

## Epic B — Multi-exchange spot markets (P1, done)

| ID | Task | Status |
|---|---|---|
| MKT-1 | RTK market endpoints in libs/api | done |
| MKT-2 | Markets page shell + ExchangeTabs (Ant) | done |
| MKT-3 | MarketsTable (Ant Table) + formatters | done |
| MKT-4 | Toolbar filters (search, quote, tags, sort) | done |
| MKT-5 | Pagination + URL query sync | done |
| MKT-6 | Live poll + visibility pause | done |
| MKT-7 | Empty/error UX + tests | done |

## Epic C — Mobile project initialization (P0 mobile, **next**)

**Plan:** `docs/design/mobile-project-initialization.md`  
**System design:** `docs/design/mobile-system-design.md`  
**Epic:** `epics/mobile-project-initialization.md`

| ID | Task | Status |
|---|---|---|
| MINIT-1 | Scaffold React Native CLI TypeScript app (no Expo) | done |
| MINIT-2 | Lint, format, Jest, path aliases | done |
| MINIT-3 | libs + Atomic + modules skeleton + boundary ESLint | done |
| MINIT-4 | libs/api store + RTK baseApi + env | done |
| MINIT-5 | OpenAPI codegen → libs/api/generated | done |
| MINIT-6 | Navigation shell + AppState hook + providers | done |
| MINIT-7 | Color tokens (match frontend) + core atoms + ScreenTemplate | done |
| MINIT-8 | Home + Markets stub pages with ViewModels | done |
| MINIT-9 | Package docs + root AGENTS/README + changelog | done |

**Suggested order:** MINIT-1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9  
**MR grouping:** (1–3) · (4–5) · (6–7) · (8–9)

## Later (not started)

| ID | Task | Status |
|---|---|---|
| DET-1 | Coin detail page + Lightweight Charts candle view (web) | backlog |
| DET-2 | RSI/EMA series on/near chart (web) | backlog |
| WL-1 | Watchlist UI (web) | backlog |
| MMKT-1 | Mobile multi-exchange spot markets list | backlog |
| MDET-1 | Mobile coin detail + charts | backlog |

## Status legend

`todo` · `in_progress` · `blocked` · `done` · `backlog`
