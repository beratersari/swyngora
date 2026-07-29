# MLEAD-1: Constants + leaderboard query helpers + tests

| Field | Value |
|---|---|
| **ID** | MLEAD-1 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-1.md` |

## Summary

Pure helpers + constants (no UI):

- `config/leaderboardConstants.ts` — page size, poll ms, default exchange/quote, board kinds, quote options  
- `libs/utils/leaderboardQuery.ts` —  
  - `LeaderboardKind = 'gainers' | 'losers' | 'volume'`  
  - `buildLeaderboardSpotQuery({ board, exchange, quote, limit, offset })`  
  - `rankLabel(offset, index)`  
  - map spot items → row view fields (reuse formatters; optional rank prefix)  
- Refactor Home movers/volume builders to call shared helpers **without changing Home defaults** (limit 5)  
- Unit tests for each board sort/order and rank math  
- Export from `libs/utils/index.ts`

## Design

`docs/design/mobile-leaderboards.md` §3 · `MLEAD-A.md`

## Acceptance

- [x] Gainers/losers/volume produce correct `sort`/`order`  
- [x] Home teasers still use limit 5 via shared builders  
- [x] No React / navigation imports  
- [x] Tests green  
- [x] Status → done when finished  

## Status

`done`
