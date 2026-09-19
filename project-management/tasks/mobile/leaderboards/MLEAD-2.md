# MLEAD-2: Atomic segment / list chrome

| Field | Value |
|---|---|
| **ID** | MLEAD-2 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-2.md` |

## Summary

Prefer **reuse** of `chip-group`, `exchange-chips`, `market-row`, `markets-list`.

Add only what is missing under `components/` (kebab-case):

| Folder | Level | Role |
|--------|-------|------|
| `molecules/leaderboard-segment` (or ChipGroup config) | molecule | Gainers / Losers / Volume |
| optional rank on `market-row` | organism prop | `rankLabel?: string` if not already supported |

No RTK/navigation inside Atomic components. Tests for new pieces only.

## Design

`docs/design/mobile-leaderboards.md` §5

## Acceptance

- [x] Board segment single-select works  
- [x] List/row can show rank without breaking Markets  
- [x] Status → done when finished  

## Status

`done`
