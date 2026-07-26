# 002 — No `src/features/`; Atomic Design only (Option A)

| Field | Value |
|---|---|
| **Status** | accepted |
| **Date** | 2026-07-26 |
| **Package** | `frontend/` |

## Context

The product UI had both:

- `components/` (Atomic: atoms → pages)
- `features/markets/components/` (domain widgets)

API already lives in `libs/api`. Pages already own RTK. Domain widgets under `features/` duplicated “where does UI live?” without owning routes or data.

## Decision

**Option A:** Remove `src/features/`. Domain UI moves to **`components/organisms/`**. Screens stay in **`components/pages/`**. Shared non-UI stays in **`libs/`**.

| Layer | Role |
|---|---|
| atoms / molecules | Design system / generic hosts |
| organisms | Domain sections (props only) |
| pages | Routes + RTK composition |
| libs | api, hooks, utils |

## Consequences

- One UI tree; simpler imports (`@/components/organisms/...`).
- When watchlist / paper / AI appear, either grow organisms or **revisit Option B** (feature-owned pages) with an explicit decision.
- Root monorepo `AGENTS.md` still mentions optional `features/` for other clients; **product web `frontend/AGENTS.md` wins** under `frontend/`.

## Rejected alternative

**Option B** — `features/<domain>/` owns pages + domain UI; `components/` is design-system-only. Deferred until multiple product areas need isolation.
