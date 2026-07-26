# Feature: Multi-exchange spot markets (mobile)

**Status:** Implemented (Epic D)  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Already implemented  
**Epic:** `project-management/epics/mobile-multi-exchange-spot-markets.md`  
**Design plan:** `docs/design/mobile-markets-dashboard.md`  
**Local tasks:** `project-management/tasks/MMKT-*.md`

---

## 1. Problem / goal

Mobile users need the same multi-exchange spot browse experience as web: list, filter, sort, paginate, live refresh — optimized for **phone-width Chrome** and RN primitives (no Ant Design).

## 2. Behavior

1. Open **Markets** tab (primary dashboard for market data).
2. Defaults: `exchange=binance`, `quote=USDT`, `sort=quoteVolume`, `order=desc`, `limit=30` (mobile page size).
3. Flat list of rows from `GET /api/v1/market/spot`.
4. Exchange chips/tabs → refetch, reset offset.
5. Search (`q`, debounced), quote picker, tag multi-select, sort control.
6. While app/document active and Markets focused, poll ~10s.
7. Pull-to-refresh forces refetch.
8. Row tap → **later** coin detail (out of this epic; may show disabled/toast in MMKT).

## 3. APIs

Same as web feature doc: exchanges, tags, spot (OpenAPI).

## 4. Where the code will live

| Area | Path |
|------|------|
| Page + VM | `mobile/src/modules/markets/pages/MarketsPage/` |
| Module UI | `mobile/src/modules/markets/components/` (flat) |
| RTK | `mobile/src/libs/api/endpoints/marketApi.ts` |
| Formatters | `mobile/src/libs/utils/` |
| Design | `docs/design/mobile-markets-dashboard.md` |

## 5. How to verify

```bash
cd mobile && npm run web
# backend: go run ./cmd/server
npm test
```
