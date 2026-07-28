# MDET-5: RSI/EMA indicator organisms and series mapping

| Field | Value |
|---|---|
| **ID** | MDET-5 |
| **Epic** | mobile-coin-detail |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/detail/MDET-5.md` |
| **Blocked by** | MDET-4 |

## Summary

GET /indicators with same interval/limit. RSI pane organism (0–100, 30/70 bands). EMA overlays on price chart (12/26). Pure mappers in helpers + tests. No batch API.

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
