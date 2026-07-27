# MWL-2: RTK watchlistApi endpoints

| Field | Value |
|---|---|
| **ID** | MWL-2 |
| **Epic** | mobile-watchlist |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/watchlist/MWL-2.md` |

## Summary

Add `libs/api/endpoints/watchlistApi.ts` with OpenAPI-typed endpoints:

- `getWatchlist` (GET)
- `addWatchlistItem` (POST)
- `removeWatchlistItem` (DELETE)
- `replaceWatchlist` (PUT)

Wire `providesTags` / `invalidatesTags` using existing `Watchlist` tag on `baseApi`. Export hooks from `libs/api/index.ts`. Prefer `X-Client-Id` header; if CORS blocks it, use query/body `clientId` (API supports both — document choice in README).

No page UI.

## Design

`docs/design/mobile-watchlist.md` §3, §8, §13 CORS note

## Acceptance

- [ ] Endpoints match OpenAPI operations
- [ ] Types from `generated/schema` (no hand-rolled DTO drift)
- [ ] Tag invalidation on mutations
- [ ] Smoke import from store
- [ ] Status updated here + `board.md`

## Status

`done`
