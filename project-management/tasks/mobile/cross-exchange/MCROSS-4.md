# MCROSS-4: Wire section into CoinDetailPage + navigate

| Field | Value |
|---|---|
| **ID** | MCROSS-4 |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-4.md` |

## Summary

- Render `CrossExchangeCompare` on `CoinDetailPage` (below stats, before chart — per design).  
- Pass ViewModel fields from MCROSS-3.  
- On row press (non-source): `navigation.navigate(MarketsScreens.Detail, { exchange, symbol })` (or replace if same stack — avoid deep stack loops; prefer navigate with new params).  
- Source row may be non-pressable or no-op.  
- Works from Markets, Favorites, and Pumps detail entry points (same page component).

## Design

`docs/design/mobile-cross-exchange-compare.md` §5–6

## Acceptance

- [x] Section visible on detail for major pairs  
- [x] Switching venue updates full detail context  
- [x] Page test covers section presence with injected VM  
- [x] Status → done when finished  

## Status

`done`
