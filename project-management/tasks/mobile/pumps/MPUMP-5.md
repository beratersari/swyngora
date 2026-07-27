# MPUMP-5: Coin detail pump events section

| Field | Value |
|---|---|
| **ID** | MPUMP-5 |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile |

## Summary

On `coin-detail-page`, add section using `useGetPumpEventsQuery` (or pass data from detail ViewModel):

- Defaults from MPUMP-A detail table  
- Render `pump-event-list`  
- Section loading/error independent of charts  
- Optional: link “See market pumps” → Pumps tab  

## Design

`docs/design/mobile-pumps.md` §5.2

## Acceptance

- [ ] Events show for symbol/exchange  
- [ ] Does not break existing detail tests  
- [ ] Status updated  

## Status

`done`
