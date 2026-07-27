# MPUMP-2: formatPump + pumpQuery helpers and unit tests

| Field | Value |
|---|---|
| **ID** | MPUMP-2 |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile |

## Summary

- `libs/utils/formatPump.ts` — format signed return %, change tone, volume ratio label, mode labels  
- `libs/utils/pumpQuery.ts` — `buildScanQuery`, `buildDetailPumpQuery` with mobile defaults from `config/pumpConstants.ts`  
- Coinbase quote default `USD` when building scan args  
- Unit tests for all pure helpers  

## Design

`docs/design/mobile-pumps.md` §3–4 · defaults in MPUMP-A §6

## Acceptance

- [ ] Defaults match design table  
- [ ] Tests green  
- [ ] Status updated  

## Status

`done`
