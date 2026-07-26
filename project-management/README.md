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
└── mobile/init|markets|detail     # MINIT-*, MMKT-*, MDET-*
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

### Mobile designs

| Doc | Role |
|---|---|
| `docs/design/mobile-project-initialization.md` | Init plan |
| `docs/design/mobile-system-design.md` | Architecture |
| `docs/design/mobile-markets-dashboard.md` | Markets dashboard |
| `docs/design/mobile-coin-detail.md` | Coin detail + indicators |

GitLab: when MCP auth works, mirror these epics/issues (see `docs/pm/`).

**Last updated:** 2026-07-26
