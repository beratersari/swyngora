# Feature: Pump / dump radar (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing `GET /api/v1/market/pumps` + `/pumps/scan` (no new endpoints)  
**Epic:** `project-management/epics/mobile-pumps.md`  
**Design:** `docs/design/mobile-pumps.md`  
**Analysis:** `project-management/tasks/mobile/pumps/MPUMP-A.md`  
**Tasks:** `project-management/tasks/mobile/pumps/MPUMP-*.md`  
**MCP (already):** `detect_pump_events`, `scan_pump_events`

---

## 1. Problem / goal

Mobile users need a fast way to see which top-volume pairs **pumped or dumped** recently, and to inspect event history on a single symbol — without leaving the app or guessing from 24h % alone.

---

## 2. Behavior (happy path)

### Scan (Pumps tab)

1. User opens **Pumps** tab.  
2. App calls `GET /api/v1/market/pumps/scan` with defaults (binance, USDT, 15m, 24h lookback, ≥8%, direction up).  
3. Ranked hits show symbol, best return %, event count.  
4. User adjusts exchange / threshold / lookback / direction → refetch.  
5. Tap hit → **Coin detail** for that pair.  
6. Pull-to-refresh re-runs scan. **No auto-poll** (scan is expensive).

### Detail events

1. On Coin detail, section loads `GET /api/v1/market/pumps?symbol=&exchange=&…`.  
2. Timeline lists events with signed `returnPct`, time, optional volume ratio.  
3. Disclaimer always visible.

### Empty / error

- No hits: suggest lower threshold or longer lookback.  
- Network/502: retry.  
- Invalid interval: fall back to venue default from `/intervals`.

---

## 3. APIs

| Method | Path |
|--------|------|
| `GET` | `/api/v1/market/pumps` |
| `GET` | `/api/v1/market/pumps/scan` |
| `GET` | `/api/v1/market/intervals` (existing — validate intervals) |
| `GET` | `/api/v1/market/exchanges` (existing) |

Field matrix: **MPUMP-A**.

---

## 4. Where code will live

| Area | Path |
|------|------|
| RTK | `mobile/src/libs/api/endpoints/pumpApi.ts` |
| Helpers | `mobile/src/libs/utils/formatPump.ts`, `pumpQuery.ts` |
| Module | `mobile/src/modules/pumps/` |
| Page | `modules/pumps/pages/pumps-scan-page/` |
| UI | `components/organisms/pump-*`, `molecules/pump-return-badge` |
| Detail | section in `modules/markets/pages/coin-detail-page/` |
| Nav | `app/navigation/RootNavigator.tsx` — **Pumps** tab |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Pumps tab → hits → detail → events section
npm test
```

---

## 6. Known limitations / follow-ups

- Mechanical thresholds only — not predictive signals.  
- Scan latency scales with `symbolLimit` (max 40).  
- OpenAPI event schemas are loose; client types follow handler DTOs (MPUMP-A).  
- Web product (`frontend/`) pumps UI is a separate backlog item.  
- Alerts on pump conditions are out of scope.
