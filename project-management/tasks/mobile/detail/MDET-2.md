# MDET-2: Navigation: CoinDetail route and markets row press

| Field | Value |
|---|---|
| **ID** | MDET-2 |
| **Epic** | mobile-coin-detail |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/detail/MDET-2.md` |
| **Blocked by** | MDET-1 recommended |

## Summary

Add MarketsScreens.Detail params { exchange, symbol } to markets stack. Wire MarketRow / MarketsPage onPressRow to navigate. Optional: interval query later.

## Design

`docs/design/mobile-coin-detail.md`  
Web field specs: `tasks/frontend/detail/DET-A.md`, `DET-B.md`

## Acceptance

- Matches design section for this task  
- UI only via `src/components/` Atomic layers (no `modules/*/components`)  
- Tests where logic is non-trivial  
- Update status here + `board.md` when done  

## Status

Update status in this file and in `project-management/board.md` when work starts/finishes.
