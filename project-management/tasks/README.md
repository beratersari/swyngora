# Local tasks layout

Tasks are grouped so the folder stays navigable as the monorepo grows.

```text
tasks/
├── README.md                 # this file
├── frontend/
│   ├── init/                 # INIT-*  (web scaffold)
│   ├── markets/              # MKT-*   (web spot markets)
│   └── detail/               # DET-*   (web coin detail + indicators)
└── mobile/
    ├── init/                 # MINIT-* (mobile scaffold)
    ├── markets/              # MMKT-*  (mobile markets dashboard)
    ├── detail/               # MDET-*  (mobile coin detail)
    ├── watchlist/            # MWL-*   (mobile watchlist / favorites)
    ├── pumps/                # MPUMP-* (mobile pump / dump radar)
    ├── batch-indicators/     # MBIND-* (mobile batch RSI/EMA list enrichment)
    ├── ai-chat/              # MAI-*   (mobile AI assistant chat)
    ├── home/                 # MHOME-* (mobile home dashboard)
    └── category-discovery/   # MCAT-*  (mobile category / tag discovery)
```

## Rules

1. **One file per task**, ID in the filename: `MDET-3.md`.
2. **Put new tasks in the matching subfolder** (surface × epic). Do not dump into `tasks/` root.
3. **Epic files** live under `project-management/epics/` and list task IDs + relative paths.
4. **Board** (`../board.md`) is the status overview; keep task files in sync when status changes.
5. Prefer short task files (summary + acceptance). Long analysis can stay as `DET-A` / `DET-B` style under the same folder.

## Adding a task

```bash
# example
cat > project-management/tasks/mobile/detail/MDET-1.md <<'TASK'
# MDET-1: …
TASK
```

Then update `board.md` and the epic checklist.
