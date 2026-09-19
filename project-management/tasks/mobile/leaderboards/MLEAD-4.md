# MLEAD-4: LeaderboardsPage View + navigation routes

| Field | Value |
|---|---|
| **ID** | MLEAD-4 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-4.md` |

## Summary

- View: ScreenTemplate + segment + exchange/quote chips + list (ViewModel props only)  
- Route: `Leaderboards` with optional params `{ board?: 'gainers' | 'losers' | 'volume' }`  
- Register on **Home stack** and/or **Markets stack** so deep links work from both tabs  
- Initialize board from route params on mount  
- Export page + screen constants from module `index` / `navigation`

## Design

`docs/design/mobile-leaderboards.md` §4

## Acceptance

- [x] Route reachable; param selects initial board  
- [x] Page test with injected ViewModel  
- [x] Status → done when finished  

## Status

`done`
