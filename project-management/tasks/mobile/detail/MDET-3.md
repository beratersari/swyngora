# MDET-3: CoinDetailPage shell + header/stats organisms + ViewModel

| Field | Value |
|---|---|
| **ID** | MDET-3 |
| **Epic** | mobile-coin-detail |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/detail/MDET-3.md` |
| **Blocked by** | MDET-2 |

## Summary

Create CoinDetailPage (View + ViewModel). Organisms: CoinDetailHeader, CoinDetailStats (or StatTile molecules). Load ticker + supply; soft-handle supply 404. Loading/error per section.

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
