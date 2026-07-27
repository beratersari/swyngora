# MWL-5: StarButton + Markets / Detail wiring

| Field | Value |
|---|---|
| **ID** | MWL-5 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-5.md` |

## Summary

- Molecule `components/molecules/StarButton/` (filled/outline, a11y labels, loading optional)
- Extend `MarketRow` with optional `watched` + `onStarPress` (star hit target must **not** trigger row navigation)
- Extend `CoinDetailHeader` (or detail page chrome) with star toggle
- Wire from MarketsPage / CoinDetailPage ViewModels or thin page handlers via `useWatchlist()` context

## Design

`docs/design/mobile-watchlist.md` §5.1

## Acceptance

- [ ] Star toggles membership without opening detail
- [ ] Detail header star stays in sync with Markets
- [ ] Atomic-only UI under `src/components/`
- [ ] Basic component/page tests
- [ ] Status updated here + `board.md`

## Status

`done`
