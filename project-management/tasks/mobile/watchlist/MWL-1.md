# MWL-1: clientId + local storage helpers

| Field | Value |
|---|---|
| **ID** | MWL-1 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-1.md` |

## Summary

Implement device-local identity and persistence utilities (no UI):

- `libs/utils/clientId.ts` — `getOrCreateClientId()` → `mobile-<uuid-v4>`; reject empty/`default`
- Storage adapter for web `localStorage` (interface ready for AsyncStorage later)
- Keys in `config/watchlistConstants.ts` (clientId key, watchlist cache key, `MAX_WATCHLIST_ITEMS = 200`)
- Persist/read minimal local watchlist array `{ exchange, symbol }[]`

## Design

`docs/design/mobile-watchlist.md` §4

## Acceptance

- [ ] UUID clientId created once and reused across reloads (unit test with mock storage)
- [ ] Never returns empty or `"default"`
- [ ] Local watchlist read/write helpers
- [ ] Unit tests green
- [ ] Status updated here + `board.md` when done

## Status

`done`
