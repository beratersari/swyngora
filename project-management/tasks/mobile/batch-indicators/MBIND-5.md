# MBIND-5: Markets list batch RSI enrichment (P1)

| Field | Value |
|---|---|
| **ID** | MBIND-5 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-1, MBIND-2, MBIND-3 |

## Summary

Enrich **Markets** visible rows with latest RSI via batch:

- Symbols = current page `rows` (or first page only to limit cost)
- Same exchange as markets filter
- Join into `MarketRowViewModel` RSI props
- Poll carefully: only when focused + active; avoid stacking with heavy spot poll — prefer slower interval than spot (e.g. 30–60s) or refresh on pull-to-refresh only for v1
- Document choice in design/changelog

## Design

MBIND-A §4 (P1 Markets) · design § Markets flow

## Acceptance

- [ ] First-page (or visible) rows show RSI when batch succeeds  
- [ ] Pagination does not fire unbounded concurrent batches without control  
- [ ] Pull-to-refresh revalidates batch  
- [ ] Markets ViewModel tests updated  
- [x] Status → done  

## Status

`done`
