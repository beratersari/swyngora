# MPUMP-4: PumpsScanPage ViewModel + View + Pumps tab navigation

| Field | Value |
|---|---|
| **ID** | MPUMP-4 |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile |

## Summary

- Module `modules/pumps/` with `pages/pumps-scan-page/` (View + ViewModel)  
- Wire `useScanPumpEventsQuery` with filter state  
- Add **Pumps** bottom tab + stack (`PumpsScan`, reuse `CoinDetail`)  
- Pull-to-refresh; **no polling**  
- Disclaimer footer  

## Design

`docs/design/mobile-pumps.md` §4–5 navigation

## Acceptance

- [ ] Tab visible; scan list works  
- [ ] Row → detail with exchange/symbol  
- [ ] ViewModel has no JSX  
- [ ] Status updated  

## Status

`done`
