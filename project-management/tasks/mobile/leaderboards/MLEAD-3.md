# MLEAD-3: LeaderboardsPage ViewModel

| Field | Value |
|---|---|
| **ID** | MLEAD-3 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-3.md` |

## Summary

ViewModel for leaderboards page:

- State: `board`, `exchange`, `quote`, `offset` / accumulated rows (Markets-style reset on filter change)  
- RTK: `useListSpotMarketsQuery` with MLEAD-1 args; poll first page when focused + AppState active  
- Map rows with rank labels  
- Optional: batch RSI enrichment (copy Markets pattern; skip if time-boxed — note in MR)  
- Handlers: select board/exchange/quote, load more, refresh, retry, press row → detail  

Prefer module ownership: **`modules/markets/pages/leaderboards-page/`** (market data) or **`modules/app/pages/leaderboards-page/`** if Home-stack-only — pick one in implementation and document in AGENTS.

## Design

`docs/design/mobile-leaderboards.md` §6

## Acceptance

- [x] Board switch resets list and refetches  
- [x] Pagination appends ranks correctly  
- [x] Poll pauses when unfocused / backgrounded  
- [x] Types in `*.types.ts`  
- [x] Status → done when finished  

## Status

`done`
