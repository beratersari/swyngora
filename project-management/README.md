# Project management (local)

Local task tracking for Swyngora until GitLab MCP/issues are fully wired.  
**Source of truth for agent work order** in this folder; keep in sync with design docs under `docs/`.

| Path | Purpose |
|---|---|
| `board.md` | Status overview (todo / in progress / done) for frontend **and** mobile |
| `epics/` | Epic definitions |
| `tasks/` | Individual tasks (`INIT-*`, `MKT-*`, `MINIT-*`, …) |
| `decisions/` | Stack/product decisions (ADRs-lite for client PM) |

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
| Runtime | **React Native CLI** — **no Expo** |
| UI | Atomic Design + StyleSheet; custom atoms first |
| Structure | `components/` Atomic only; **`modules/*/pages` + ViewModel** |
| Data | RTK Query + OpenAPI (`src/libs/api`) |
| **Colors** | **Same as frontend** (`navy` / `indigo` / `steel` / `cream`) |

See `decisions/002-react-native-cli-modules-viewmodel.md`.

## Work order

1. Epic A — Frontend project initialization (`epics/frontend-project-initialization.md`) — **done**
2. Epic B — Multi-exchange spot markets (`epics/multi-exchange-spot-markets.md`) — **done**
3. **Epic C — Mobile project initialization** (`epics/mobile-project-initialization.md`) — **next**
4. Later — mobile markets / detail (`MMKT-*`, `MDET-*` on board)

### Mobile init designs

| Doc | Role |
|---|---|
| `docs/design/mobile-project-initialization.md` | Init plan (checklist, tasks, acceptance) |
| `docs/design/mobile-system-design.md` | Full architecture (modules, ViewModel, nav, PR plan) |

GitLab: when MCP auth works, mirror these epics/issues (see `docs/pm/`).

**Last updated:** 2026-07-26
