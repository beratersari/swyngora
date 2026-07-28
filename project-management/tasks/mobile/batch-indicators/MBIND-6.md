# MBIND-6: Loading / partial failure / disclaimer UX

| Field | Value |
|---|---|
| **ID** | MBIND-6 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-4 (and MBIND-5 if markets landed) |

## Summary

Harden UX across Favorites (+ Markets if present):

- Row-level RSI loading vs list-level price loading (do not block list on batch)
- Banner for whole-batch failure (400/429/502) with Retry
- Per-item unavailable silent `—`
- Informational disclaimer (not financial advice) once per screen
- Optional: document rate-limit / backoff behavior

## Acceptance

- [ ] Batch failure does not empty the favorites/markets list  
- [ ] Retry path works  
- [ ] Disclaimer visible  
- [ ] Manual checklist in task notes or feature doc  
- [x] Status → done  

## Status

`done`
