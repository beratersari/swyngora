# Product client board

Local status for web (`frontend/`) and mobile (`mobile/`) until GitLab issues are fully wired.

**Tasks live under subfolders** — see `tasks/README.md` (do not add new files to `tasks/` root).

## Frontend stack (locked)

- UI: **Ant Design**
- Charts: **TradingView Lightweight Charts**
- Data: RTK Query + OpenAPI in `libs/api`
- Colors: `frontend/src/styles/tokens/colors.ts`

## Mobile stack (locked)

- Runtime: **React Native** (**no Expo**); Chrome via **react-native-web**
- UI: **Atomic Design only** under `src/components/` (no `modules/*/components`)
- Architecture: **`modules/*/pages` + ViewModel + module context**
- Data: RTK Query + OpenAPI in `libs/api`
- Colors: same tokens as frontend
- Decision: `decisions/002-react-native-cli-modules-viewmodel.md`

---

## Epic A — Frontend project initialization (P0, done)

**Tasks:** `tasks/frontend/init/`

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

**Tasks:** `tasks/frontend/markets/`

| ID | Task | Status |
|---|---|---|
| MKT-1 | RTK market endpoints in libs/api | done |
| MKT-2 | Markets page shell + ExchangeTabs (Ant) | done |
| MKT-3 | MarketsTable (Ant Table) + formatters | done |
| MKT-4 | Toolbar filters (search, quote, tags, sort) | done |
| MKT-5 | Pagination + URL query sync | done |
| MKT-6 | Live poll + visibility pause | done |
| MKT-7 | Empty/error UX + tests | done |

## Epic — Frontend coin detail (done)

**Tasks:** `tasks/frontend/detail/` · Feature: `docs/features/coin-detail.md`

| ID | Task | Status |
|---|---|---|
| DET-A | Analysis: detail APIs field matrix | done |
| DET-B | Analysis: indicators field matrix | done |
| DET-1 | RTK detail endpoints | done |
| DET-2 | Page shell + route + header/stats | done |
| DET-3 | Candles + toolbar | done |
| DET-4 | Indicator panel + overlays + tests | done |

## Epic C — Mobile project initialization (P0, done — MR !15)

**Tasks:** `tasks/mobile/init/`  
**Plan:** `docs/design/mobile-project-initialization.md`

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

## Epic D — Mobile multi-exchange spot markets / dashboard (done)

**Tasks:** `tasks/mobile/markets/`  
**Plan:** `docs/design/mobile-markets-dashboard.md`  
**Branch:** `feature/mobile-spot-markets`

| ID | Task | Status |
|---|---|---|
| MMKT-1 | RTK marketApi: exchanges, tags, spot | done |
| MMKT-2 | Formatters + spot query helpers | done |
| MMKT-3 | ExchangeChips + MarketsFilterBar | done |
| MMKT-4 | MarketRow + MarketsList | done |
| MMKT-5 | MarketsPage ViewModel (filters, poll, pagination) | done |
| MMKT-6 | MarketsPage View + pull-to-refresh | done |
| MMKT-7 | Empty/error/loading UX + tests | done |
| MMKT-8 | Docs + board + changelog closeout | done |

## Epic E — Mobile coin detail + indicators (**done**)

**Tasks:** `tasks/mobile/detail/`  
**Epic:** `epics/mobile-coin-detail.md`  
**Plan:** `docs/design/mobile-coin-detail.md`  
**Feature:** `docs/features/mobile-coin-detail.md`  
**Branch:** `feature/mobile-coin-detail` (after / stacked on markets)

| ID | Task | Status |
|---|---|---|
| MDET-1 | RTK detail endpoints (intervals, ticker, supply, candles, indicators) | done |
| MDET-2 | Navigation: CoinDetail route + markets row press | done |
| MDET-3 | CoinDetailPage shell + header/stats organisms + ViewModel | done |
| MDET-4 | Interval toolbar + candle chart organism | done |
| MDET-5 | RSI/EMA indicator organisms + series mapping | done |
| MDET-6 | Section loading/error, polling pause, tests | done |
| MDET-7 | Docs + board + changelog closeout | done |

**MR grouping:** (1–2) · (3) · (4–5) · (6–7)

## Epic F — Mobile watchlist (**done**)

**Tasks:** `tasks/mobile/watchlist/`  
**Epic:** `epics/mobile-watchlist.md`  
**Plan:** `docs/design/mobile-watchlist.md`  
**Feature:** `docs/features/mobile-watchlist.md`  
**Branch:** `feature/mobile-watchlist`

| ID | Task | Status |
|---|---|---|
| MWL-1 | clientId + local storage helpers | done |
| MWL-2 | RTK watchlistApi endpoints | done |
| MWL-3 | watchKey + merge pure helpers + tests | done |
| MWL-4 | WatchlistProvider context (hydrate + toggle) | done |
| MWL-5 | StarButton + Markets / Detail wiring | done |
| MWL-6 | Watchlist tab + WatchlistPage + list organisms | done |
| MWL-7 | Quote enrichment + poll + empty/error UX | done |
| MWL-8 | Tests polish + docs/board/changelog closeout | done |

**MR grouping:** (1–3) · (4–5) · (6–7) · (8)

## Epic G — Mobile pump / dump radar (**done**)

**Tasks:** `tasks/mobile/pumps/`  
**Epic:** `epics/mobile-pumps.md`  
**Plan:** `docs/design/mobile-pumps.md`  
**Feature:** `docs/features/mobile-pumps.md`  
**Analysis:** `tasks/mobile/pumps/MPUMP-A.md`  
**Branch:** `feature/mobile-pumps`

| ID | Task | Status |
|---|---|---|
| MPUMP-A | API field matrix analysis | done |
| MPUMP-1 | RTK pumpApi endpoints (scan + get events) | done |
| MPUMP-2 | formatPump + pumpQuery helpers and tests | done |
| MPUMP-3 | Pump Atomic UI (badge, filters, hit/event lists) | done |
| MPUMP-4 | PumpsScanPage + Pumps tab navigation | done |
| MPUMP-5 | Coin detail pump events section | done |
| MPUMP-6 | Loading / empty / error / disclaimer | done |
| MPUMP-7 | Tests polish + docs/board/changelog closeout | done |

**MR grouping:** (A+1–2) · (3–4) · (5–6) · (7)

## Epic H — Mobile batch indicators (**done**)

**Tasks:** `tasks/mobile/batch-indicators/`  
**Epic:** `epics/mobile-batch-indicators.md`  
**Plan:** `docs/design/mobile-batch-indicators.md`  
**Feature:** `docs/features/mobile-batch-indicators.md`  
**Analysis:** `tasks/mobile/batch-indicators/MBIND-A.md`  
**Branch:** `feature/mobile-batch-indicators`

| ID | Task | Status |
|---|---|---|
| MBIND-A | API field matrix analysis | done |
| MBIND-1 | RTK postIndicatorsBatch endpoint | done |
| MBIND-2 | Chunk/group/format helpers + tests | done |
| MBIND-3 | RSI badge + row prop extensions | done |
| MBIND-4 | Favorites batch RSI enrichment (P0) | done |
| MBIND-5 | Markets list batch RSI enrichment (P1) | done |
| MBIND-6 | Loading / partial failure / disclaimer | done |
| MBIND-7 | Tests polish + docs/board/changelog closeout | done |

**MR grouping:** (A+1–2) · (3–4) · (5–6) · (7)


## Epic I — Mobile AI assistant chat (**done**)

**Tasks:** `tasks/mobile/ai-chat/`  
**Epic:** `epics/mobile-ai-chat.md`  
**Plan:** `docs/design/mobile-ai-chat.md`  
**Feature:** `docs/features/mobile-ai-chat.md`  
**Analysis:** `tasks/mobile/ai-chat/MAI-A.md`  
**Branch:** `feature/mobile-ai-chat`

| ID | Task | Status |
|---|---|---|
| MAI-A | API field matrix analysis | done |
| MAI-1 | OpenAPI for AI chat + client codegen | done |
| MAI-2 | RTK aiApi chat mutation | done |
| MAI-3 | sessionId + message model helpers | done |
| MAI-4 | Atomic chat UI (bubbles, composer, list) | done |
| MAI-5 | Ask tab + AiChatPage ViewModel | done |
| MAI-6 | Context chips from Markets / Detail / Pumps | done |
| MAI-7 | Loading / 503 / error / disclaimer UX | done |
| MAI-8 | Docs + board + changelog closeout | done |

**MR grouping:** (A+1) · (2–3) · (4–5) · (6–7) · (8)


## Epic J — Mobile home dashboard (**done**)

**Tasks:** `tasks/mobile/home/`  
**Epic:** `epics/mobile-home-dashboard.md`  
**Plan:** `docs/design/mobile-home-dashboard.md`  
**Feature:** `docs/features/mobile-home-dashboard.md`  
**Branch:** `feature/mobile-home-dashboard`

| ID | Task | Status |
|---|---|---|
| MHOME-1 | Dashboard constants + query helpers | done |
| MHOME-2 | Atomic dashboard UI | done |
| MHOME-3 | HomePage ViewModel | done |
| MHOME-4 | HomePage View + deep links | done |
| MHOME-5 | Empty / partial failure / loading UX | done |
| MHOME-6 | Tests | done |
| MHOME-7 | Docs + board + changelog closeout | done |

**MR grouping:** (1–2) · (3–4) · (5–6) · (7)

## Later (not started)

| ID | Task | Status |
|---|---|---|
| WL-1 | Watchlist UI (web / `frontend/`) | backlog |

## Status legend

`todo` · `in_progress` · `blocked` · `done` · `backlog`
