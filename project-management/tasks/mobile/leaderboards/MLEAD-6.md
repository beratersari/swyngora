# MLEAD-6: Loading / empty / error / poll / i18n

| Field | Value |
|---|---|
| **ID** | MLEAD-6 |
| **Epic** | mobile-leaderboards |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/leaderboards/MLEAD-6.md` |

## Summary

| State | Behavior |
|-------|----------|
| Initial load | List skeleton / Markets loading pattern |
| Empty | Board-specific empty copy |
| Error | Message + retry |
| Pull-to-refresh | Refetch first page |
| Poll paused caption | Optional caption when backgrounded |
| i18n | en + tr — new `leaderboards` ns and/or `home`/`markets` keys |

No hard-coded user copy in View.

## Design

`docs/design/mobile-leaderboards.md` §8

## Acceptance

- [x] en/tr parity for new strings  
- [x] Empty/error paths covered  
- [x] Status → done when finished  

## Status

`done`
