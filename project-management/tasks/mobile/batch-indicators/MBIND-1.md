# MBIND-1: RTK batch indicators endpoint

| Field | Value |
|---|---|
| **ID** | MBIND-1 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-A |

## Summary

Add `postIndicatorsBatch` to mobile RTK (`marketApi` inject or dedicated `indicatorsApi`):

- `POST /api/v1/market/indicators/batch`
- Body: `{ exchange?, interval?, symbols[], rsiPeriod?, emaPeriods? }`
- Types aligned with OpenAPI + MBIND-A (handler DTOs)
- Prefer **query** with body + `serializeQueryArgs` for cacheability
- Register tags (e.g. `Indicator` / batch id); export hooks from `libs/api/index.ts`
- No page UI in this task

## Design

`docs/design/mobile-batch-indicators.md` · MBIND-A §5

## Acceptance

- [ ] Endpoint + typed request/response  
- [ ] Optional fields omitted cleanly  
- [ ] `providesTags` + cache key stable for same symbol set  
- [ ] Exported hooks  
- [ ] Unit smoke test for arg builder if colocated  
- [x] Status → done  

## Status

`done`
