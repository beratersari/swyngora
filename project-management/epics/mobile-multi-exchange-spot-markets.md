# Epic D: Mobile multi-exchange spot markets (dashboard)

**Priority:** P0 (mobile track, after init)  
**Status:** todo  
**Blocks:** mobile coin detail  
**Depends on:** Epic C mobile init (MR !15 / `feature/mobile-init`)  
**Design plan:** `docs/design/mobile-markets-dashboard.md`  
**Feature note:** `docs/features/mobile-multi-exchange-spot-markets.md`  
**Branch:** `feature/mobile-spot-markets`

## Goal

Ship the primary **mobile markets dashboard** in Chrome (react-native-web): browse Binance / Coinbase / Bybit spot markets with search, filters, sort, pagination, and live refresh — same APIs as web Epic B, mobile UX (list + filters, View + ViewModel).

## APIs

- `GET /api/v1/market/exchanges`
- `GET /api/v1/market/tags`
- `GET /api/v1/market/spot`

## Tasks

- [ ] MMKT-1 … MMKT-8 (see `project-management/board.md` and `tasks/MMKT-*.md`)

## Acceptance

- Markets tab shows real spot rows (not placeholder)
- Exchange switch, search, quote, tags, sort, pagination work
- Polling pauses when tab/document inactive
- View has no RTK imports; ViewModel owns queries
- Works in Chrome via `npm run web`
- Tests for VM mapping + formatters + empty/error UX
