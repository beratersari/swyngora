# MHOME-1: Dashboard constants + query helpers

| Field | Value |
|---|---|
| **ID** | MHOME-1 |
| **Epic** | mobile-home-dashboard |
| **Status** | todo |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/home/MHOME-1.md` |

## Summary

Pure helpers + constants (no UI):

- `config/homeDashboardConstants.ts` — page sizes (e.g. movers 5, volume 5, pumps 3), poll intervals, default exchange/quote
- `libs/utils/homeDashboardQuery.ts` — builders for spot movers/volume args and pump scan teaser args
- Map `SpotMarket` → compact row view fields (reuse formatters)
- Unit tests for query builders and mappers

## Design

`docs/design/mobile-home-dashboard.md` §3–4

## Acceptance

- [ ] Stable query args for RTK cache keys  
- [ ] No React / navigation imports  
- [ ] Tests green  
- [ ] Status → done when finished  

## Status

`todo`
