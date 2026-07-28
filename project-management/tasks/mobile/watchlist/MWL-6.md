# MWL-6: Watchlist tab + WatchlistPage + list organisms

| Field | Value |
|---|---|
| **ID** | MWL-6 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-6.md` |

## Summary

- Add **Watchlist** bottom tab + stack (`WatchlistList`, reuse `CoinDetailPage`)
- `modules/watchlist/pages/WatchlistPage/` View + ViewModel
- Organisms: `WatchlistRow`, `WatchlistList`
- Empty state with CTA toward Markets
- Row press → Coin detail params `{ exchange, symbol }`
- Unstar from row

Quotes can be placeholders until MWL-7.

## Design

`docs/design/mobile-watchlist.md` §5.2, §6

## Acceptance

- [ ] Tab visible; list shows starred items from context
- [ ] Navigation to detail works from watchlist stack
- [ ] Empty state when no items
- [ ] ViewModel has no JSX
- [ ] Status updated here + `board.md`

## Status

`done`
