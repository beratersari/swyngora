# MCROSS-3: Coin detail ViewModel — parallel tickers + mapping

| Field | Value |
|---|---|
| **ID** | MCROSS-3 |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | todo |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-3.md` |

## Summary

Extend coin detail data layer (prefer hook colocated with detail ViewModel or pure composition inside `CoinDetailPage.viewModel.ts`):

1. Build plan from route `exchange` + `symbol` (MCROSS-1).  
2. Fetch tickers for each venue — prefer:
   - multiple `useGetTicker24hQuery` with `skip` when no symbol, **or**
   - `useLazyGetTicker24hQuery` + effect with candidate fallback on 404.  
3. Map results to `crossExchangeRows` view models.  
4. Poll only when detail focused + AppState active (match ticker poll constant).  
5. Expose `onPressExchangeRow` navigation target data (wired in MCROSS-4).

Keep RTK in libs/api only (existing ticker endpoint).

## Design

`docs/design/mobile-cross-exchange-compare.md` §5, §7

## Acceptance

- [ ] Source row always present when main ticker works  
- [ ] Candidate fallback attempted for non-source venues  
- [ ] Partial failure isolated per row  
- [ ] ViewModel unit/integration test with mocked API hooks or pure mapper tests  
- [ ] Status → done when finished  

## Status

`todo`
