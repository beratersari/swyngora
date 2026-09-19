# MLEAD-5: Home “See all” deep links + Markets entry

| Field | Value |
|---|---|
| **ID** | MLEAD-5 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-5.md` |

## Summary

**Home**

- Top movers “See all” → Leaderboards `board=gainers`  
- Highest volume “See all” → Leaderboards `board=volume`  
- (Optional) quick action chip “Leaderboards”  

**Markets**

- Toolbar or secondary entry to Leaderboards (same stack or navigate to Home stack route — keep UX simple)

Update Home ViewModel handlers; tests that navigation targets correct board when possible (mock navigation).

## Design

`docs/design/mobile-leaderboards.md` §4, §7

## Acceptance

- [x] Home movers See all opens gainers  
- [x] Home volume See all opens volume  
- [x] Markets has a discoverable entry  
- [x] Status → done when finished  

## Status

`done`
