# MWL-4: WatchlistProvider context (hydrate + toggle)

| Field | Value |
|---|---|
| **ID** | MWL-4 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-4.md` |

## Summary

Create `modules/watchlist/context/`:

- Hydrate: local cache → GET merge → optional re-POST missing server items (or PUT replace strategy documented)
- API: `items`, `isWatched(ex, sym)`, `toggle(ex, sym)`, `isReady`, `error`, `count`
- Optimistic toggle with rollback on mutation failure
- Block add at 200 with clear error message
- Provider mounted high enough for Markets + Watchlist tabs (e.g. wrap MainTabs or both stacks)

No star UI yet (or minimal if needed for tests).

## Design

`docs/design/mobile-watchlist.md` §4 WatchlistContext

## Acceptance

- [ ] Context unit/integration tests with mocked RTK
- [ ] Optimistic add/remove + rollback
- [ ] Max-items guard
- [ ] No JSX business rules outside context/VM
- [ ] Status updated here + `board.md`

## Status

`done`
