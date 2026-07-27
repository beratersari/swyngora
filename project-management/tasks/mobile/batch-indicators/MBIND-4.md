# MBIND-4: Favorites enrichment with batch RSI (P0)

| Field | Value |
|---|---|
| **ID** | MBIND-4 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-1, MBIND-2, MBIND-3 |

## Summary

Wire batch indicators into **Favorites / Watchlist**:

- From `watchlist.items`, group by exchange, chunk ≤50
- Call batch query(ies) when tab focused + AppState active
- Join RSI onto enriched rows (alongside existing ticker enrichment)
- Skip when empty list
- Pause poll in background
- Optional one-line disclaimer on Favorites (from API `note` or product constant)

Prefer extending ViewModel / page-local enrichment (pattern of `EnrichedWatchlistRow` or batch at page level — **batch at page/group level is preferred** over per-row N queries).

## Design

MBIND-A §4 (P0 Favorites) · design § Favorites flow

## Acceptance

- [ ] ≤1 batch request per exchange for normal lists (≤50)  
- [ ] RSI shows when available; `—` on item error  
- [ ] No poll when backgrounded or tab unfocused  
- [ ] Empty favorites does not call batch  
- [ ] Page / ViewModel tests with mocked RTK  
- [x] Status → done  

## Status

`done`
