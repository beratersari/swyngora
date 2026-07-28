# Feature: Batch indicators on mobile lists

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing `POST /api/v1/market/indicators/batch` (no new endpoints)  
**Epic:** `project-management/epics/mobile-batch-indicators.md`  
**Design:** `docs/design/mobile-batch-indicators.md`  
**Analysis:** `project-management/tasks/mobile/batch-indicators/MBIND-A.md`  
**Tasks:** `project-management/tasks/mobile/batch-indicators/MBIND-*.md`  

---

## 1. Problem / goal

Mobile list screens need **latest RSI** (optional EMA) for many symbols without N+1 detail indicator calls. Backend already exposes batch snapshots for table enrichment.

---

## 2. Behavior (happy path)

### Favorites (P0)

1. User opens **Favorites** with starred pairs.  
2. App groups pairs by exchange and calls `POST /api/v1/market/indicators/batch` (≤50 symbols each).  
3. Each row shows **RSI** badge when available (alongside price).  
4. Focus + AppState control refresh; background pauses batch.  
5. Pull-to-refresh revalidates membership, quotes, and indicators.

### Markets (P1)

1. Markets page loads a page of spot rows.  
2. App batches those symbols for the selected exchange.  
3. Rows show RSI; pull-to-refresh revalidates.

### Coin detail

Unchanged: full series via `GET /api/v1/market/indicators`.

### Empty / error

- Empty favorites: no batch call.  
- Whole-batch failure: banner + Retry; prices still shown.  
- Per-symbol `error: unavailable`: that row RSI `—`.  

---

## 3. APIs

| Method | Path |
|--------|------|
| `POST` | `/api/v1/market/indicators/batch` |
| `GET` | `/api/v1/market/indicators` (detail only — existing) |
| `GET` | `/api/v1/market/exchanges` / intervals as needed |

Field matrix: **MBIND-A**.

---

## 4. Where the code will live

| Area | Path |
|------|------|
| RTK | `mobile/src/libs/api/endpoints/marketApi.ts` (or `indicatorsApi.ts`) |
| Helpers | `mobile/src/libs/utils/` (`batchIndicators*`, format RSI) |
| Config | `mobile/src/config/` batch constants |
| UI | `components/molecules/rsi-badge/`, row organisms |
| Pages | `modules/watchlist/…`, `modules/markets/…` |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Star pairs → Favorites shows RSI
# Markets first page shows RSI (after MBIND-5)
npm test
```

---

## 6. Known limitations / follow-ups

- Mechanical indicators only — not trade signals.  
- Max 50 symbols per request; multi-exchange → multiple POSTs.  
- List interval fixed to `1h` in v1.  
- Web product markets RSI columns are a separate backlog.  
- Optional MCP batch tool for agents.  
