# MWL-3: watchKey + merge pure helpers

| Field | Value |
|---|---|
| **ID** | MWL-3 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-3.md` |

## Summary

Port pure logic from `simple-frontend/watchlist-logic.js` into `libs/utils/`:

- `watchKey(exchange, symbol)` → `binance|BTCUSDT` (lower exchange, upper symbol)
- `mergeWatchlists(local, server)` — union by key; **local wins** on conflict
- Optional: `isAtMaxItems(items, max = 200)`

Unit tests for merge edge cases (empty server, offline adds, case normalization).

## Design

`docs/design/mobile-watchlist.md` §4 merge policy

## Acceptance

- [ ] Helpers pure (no React/RTK)
- [ ] Tests cover merge / key / max
- [ ] Exported from `libs/utils/index.ts` if appropriate
- [ ] Status updated here + `board.md`

## Status

`done`
