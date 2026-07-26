# MDET-6: Section loading/error, polling pause, tests

| Field | Value |
|---|---|
| **ID** | MDET-6 |
| **Epic** | mobile-coin-detail |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/detail/MDET-6.md` |
| **Blocked by** | MDET-5 |

## Summary

AppState + useIsFocused pause for ticker/candle polls. Partial failure UX. Tests: helpers, ViewModel mapping, page with injected VM.

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
