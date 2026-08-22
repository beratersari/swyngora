# Feature: Coin detail + indicators (mobile)

**Status:** Implemented (Epic E)  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing market APIs (same as web coin detail)  
**Epic:** `project-management/epics/mobile-coin-detail.md`  
**Design:** `docs/design/mobile-coin-detail.md`  
**Tasks:** `project-management/tasks/mobile/detail/MDET-*.md`  
**Web parity analysis:** `docs/features/coin-detail.md`, `tasks/frontend/detail/DET-A.md`, `DET-B.md`

## Goal

Open a single-pair detail screen from the markets list: 24h ticker, supply, holders, OHLCV, RSI/EMA.

## Behavior (happy path)

1. User taps a market row.  
2. Detail shows symbol header, 24h stats, supply, holder count and top-10 share.  
3. User changes interval → candles + indicators refetch.  
4. Chart overlay chips: EMA (from loaded candles so pan-left history keeps the line), Pumps markers, Margin price lines (pump high/low).  
5. While screen focused and app active, ticker/candles poll.  
6. Back returns to markets list.

## Code homes (planned)

| Area | Path |
|------|------|
| Page | `mobile/src/modules/markets/pages/CoinDetailPage/` |
| Organisms | `mobile/src/components/organisms/…` |
| RTK | `mobile/src/libs/api/endpoints/marketApi.ts` |

## Verify

```bash
cd mobile && npm run web
# backend running; Markets → row → detail
```
