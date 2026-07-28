# MWL-7: Quote enrichment + poll + empty/error UX

| Field | Value |
|---|---|
| **ID** | MWL-7 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-7.md` |

## Summary

- Enrich watchlist rows via `useGetTicker24hQuery` (or batch of hooks / lazy queries) per `{ exchange, symbol }`
- Show last price + 24h % (reuse formatters from `libs/utils/formatPrice` / `formatMarket`)
- Poll while tab focused + `useAppStateActive()`; `pollingInterval: 0` when inactive
- Pull-to-refresh: invalidate/refetch tickers + optional GET watchlist
- Cap enrichment if list is large (see design §7)
- Error banner when membership GET fails; skeleton while first enrich loads

## Design

`docs/design/mobile-watchlist.md` §5.3, §7

## Acceptance

- [ ] Quotes render for successful tickers; “—” on failure
- [ ] Poll pauses in background
- [ ] Pull-to-refresh works
- [ ] Tests for ViewModel poll gating / enrichment skip
- [ ] Status updated here + `board.md`

## Status

`done`
