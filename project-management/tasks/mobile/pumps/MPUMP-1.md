# MPUMP-1: RTK pumpApi endpoints (scan + get events)

| Field | Value |
|---|---|
| **ID** | MPUMP-1 |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile |

## Summary

Add `libs/api/endpoints/pumpApi.ts`:

- `scanPumpEvents` → `GET /api/v1/market/pumps/scan`
- `getPumpEvents` → `GET /api/v1/market/pumps`

Wire types from MPUMP-A (handler DTO shapes). Register `'Pump'` tag on `baseApi`. Export hooks from `libs/api/index.ts`. No page UI.

## Design

`docs/design/mobile-pumps.md` §6 · `MPUMP-A.md`

## Acceptance

- [ ] Endpoints + providesTags  
- [ ] compactParams for optional query fields  
- [ ] Unit tests for any pure query builders colocated or in utils (with MPUMP-2)  
- [ ] Status updated  

## Status

`done`
