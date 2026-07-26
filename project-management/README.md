# Project management (local)

Local task tracking for Swyngora until GitLab MCP/issues are fully wired.  
**Source of truth for agent work order** in this folder; keep in sync with design docs under `docs/`.

| Path | Purpose |
|---|---|
| `board.md` | Status overview (todo / in progress / done) |
| `epics/` | Epic definitions |
| `tasks/` | Individual tasks (INIT-*, MKT-*, …) |
| `decisions/` | Stack/product decisions (ADRs-lite for frontend PM) |

## Active frontend stack (decided)

| Layer | Choice |
|---|---|
| UI kit | **Ant Design** (`antd`) |
| Charts | **TradingView Lightweight Charts** (`lightweight-charts`) |
| Data | RTK Query + OpenAPI (`src/libs/api`) |
| Layout of non-UI code | `src/libs/{api,hooks,utils}` |

See `decisions/001-antd-and-lightweight-charts.md`.

## Work order

1. Epic A — Frontend project initialization (`epics/frontend-project-initialization.md`)
2. Epic B — Multi-exchange spot markets (`epics/multi-exchange-spot-markets.md`)

GitLab: when MCP auth works, mirror these epics/issues (see `docs/pm/`).

**Last updated:** 2026-07-26
