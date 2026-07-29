# MCROSS-5: Loading / partial failure / disclaimer / i18n

| Field | Value |
|---|---|
| **ID** | MCROSS-5 |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-5.md` |

## Summary

| State | Behavior |
|-------|----------|
| Initial load | Skeleton rows for venues |
| One venue 404 after candidates | Row: unavailable |
| Network/error | Row-level error; others OK |
| All fail | Section still shows with errors (or compact message) |
| Disclaimer | Informational; not arb / not financial advice |
| i18n | en + tr in `detail` (and common if shared) |

No hard-coded user-facing English in View.

## Design

`docs/design/mobile-cross-exchange-compare.md` §8

## Acceptance

- [x] en/tr parity for new strings  
- [x] Disclaimer visible when section has data  
- [x] Status → done when finished  

## Status

`done`
