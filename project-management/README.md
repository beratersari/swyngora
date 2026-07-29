# Project management (local)

Local task tracking for Swyngora until GitLab MCP/issues are fully wired.  
**Source of truth for agent work order** in this folder; keep in sync with design docs under `docs/`.

| Path | Purpose |
|---|---|
| `board.md` | Status overview (todo / in progress / done) for frontend **and** mobile |
| `epics/` | Epic definitions |
| `tasks/` | Tasks in **subfolders by surface × epic** (see `tasks/README.md`) |
| `decisions/` | Stack/product decisions (ADRs-lite for client PM) |

## Tasks layout

```text
tasks/
├── frontend/init|markets|detail   # INIT-*, MKT-*, DET-*
└── mobile/init|markets|detail|watchlist|pumps|batch-indicators|ai-chat|home|category-discovery|cross-exchange
    # MINIT-*, MMKT-*, MDET-*, MWL-*, MPUMP-*, MBIND-*, MAI-*, MHOME-*, MCAT-*, MCROSS-*
```

**Do not add new task files to `tasks/` root.** Use the matching subfolder.

## Active frontend stack (decided)

| Layer | Choice |
|---|---|
| UI kit | **Ant Design** (`antd`) |
| Charts | **TradingView Lightweight Charts** (`lightweight-charts`) |
| Data | RTK Query + OpenAPI (`src/libs/api`) |
| Layout of non-UI code | `src/libs/{api,hooks,utils}` |
| Colors | `frontend/src/styles/tokens/colors.ts` |

See `decisions/001-antd-and-lightweight-charts.md`.

## Active mobile stack (decided)

| Layer | Choice |
|---|---|
| Runtime | **React Native** — **no Expo**; Chrome via **react-native-web** |
| UI | **Atomic Design only** under `src/components/` (no module feature components) |
| Structure | `modules/*/pages` + ViewModel + context |
| Data | RTK Query + OpenAPI (`src/libs/api`) |
| Colors | Same brand tokens as frontend |

See `decisions/002-react-native-cli-modules-viewmodel.md`.

## Work order

1. Epic A — Frontend project initialization — **done** (`tasks/frontend/init/`)
2. Epic B — Multi-exchange spot markets (web) — **done** (`tasks/frontend/markets/`)
3. Frontend coin detail — **done** (`tasks/frontend/detail/`)
4. Epic C — Mobile project initialization — **done** (`tasks/mobile/init/`)
5. Epic D — Mobile markets dashboard — **done** (`tasks/mobile/markets/`)
6. Epic E — Mobile coin detail — **done** (`tasks/mobile/detail/`, `epics/mobile-coin-detail.md`)
7. Epic F — Mobile watchlist — **done** (`tasks/mobile/watchlist/`, `epics/mobile-watchlist.md`)
8. Epic G — Mobile pump / dump radar — **done** (`tasks/mobile/pumps/`, `epics/mobile-pumps.md`)
9. Epic H — Mobile batch indicators — **done** (`tasks/mobile/batch-indicators/`, `epics/mobile-batch-indicators.md`)
10. Epic I — Mobile AI assistant chat — **done** (`tasks/mobile/ai-chat/`, `epics/mobile-ai-chat.md`)
11. Epic J — Mobile home dashboard — **done** (`tasks/mobile/home/`, `epics/mobile-home-dashboard.md`)
12. Epic K — Mobile category discovery — **done** (`tasks/mobile/category-discovery/`, `epics/mobile-category-discovery.md`)
13. Epic L — Mobile cross-exchange coin comparison — **done** (`tasks/mobile/cross-exchange/`, `epics/mobile-cross-exchange-compare.md`)

### Mobile designs

| Doc | Role |
|---|---|
| `docs/design/mobile-project-initialization.md` | Init plan |
| `docs/design/mobile-system-design.md` | Architecture |
| `docs/design/mobile-markets-dashboard.md` | Markets dashboard |
| `docs/design/mobile-coin-detail.md` | Coin detail + indicators |
| `docs/design/mobile-watchlist.md` | Watchlist (stars + tab) |
| `docs/design/mobile-pumps.md` | Pump / dump scan radar |
| `docs/design/mobile-batch-indicators.md` | Batch RSI/EMA list enrichment |
| `docs/design/mobile-ai-chat.md` | AI assistant Ask tab / chat |
| `docs/design/mobile-home-dashboard.md` | Home live widgets dashboard |
| `docs/design/mobile-category-discovery.md` | Product tag / category browse |
| `docs/design/mobile-cross-exchange-compare.md` | Cross-venue ticker compare on detail |

GitLab: when MCP auth works, mirror these epics/issues (see `docs/pm/`).

**Last updated:** 2026-07-29 (Epic L cross-exchange compare implemented)
