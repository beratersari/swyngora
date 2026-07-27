# Feature: Watchlist (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing watchlist API (no new endpoints)  
**Epic:** `project-management/epics/mobile-watchlist.md`  
**Design:** `docs/design/mobile-watchlist.md`  
**Tasks:** `project-management/tasks/mobile/watchlist/MWL-*.md`  
**Parity:** `simple-frontend/` dashboard stars; Telegram `/watch`

---

## 1. Problem / goal

Mobile users need a personal list of spot pairs across Binance / Coinbase / Bybit, with one-tap star from markets and detail, and a dedicated screen to reopen them quickly.

---

## 2. Behavior (happy path)

1. User opens **Markets**, taps **★** on a row (or stars from **Coin detail**).  
2. Pair is saved under a device **clientId** (`mobile-<uuid>`), stored locally and via `POST /api/v1/watchlist/items`.  
3. **Watchlist** tab shows all saved pairs (exchange + symbol), with 24h price/change when enrichment succeeds.  
4. Tap a watchlist row → **Coin detail** for that exchange/symbol.  
5. Unstar (row × or star off) → `DELETE` + local remove.  
6. Pull-to-refresh reloads membership + quotes.  
7. While Watchlist tab focused and app active, quotes poll on an interval; pause when backgrounded.

### Limits

- Max **200** items per clientId (backend hard cap).  
- clientId must not be empty or `default`.  
- Server store is **in-memory** — process restart clears server list; client rehydrates from local cache and re-POSTs.

### Error paths

| Case | UX |
|------|-----|
| Network fail on add | Revert star; show error |
| At max items | Block add; “Watchlist full (200)” |
| GET fails | Keep local list if present; banner |

---

## 3. APIs

| Method | Path |
|--------|------|
| `GET` | `/api/v1/watchlist` |
| `POST` | `/api/v1/watchlist/items` |
| `DELETE` | `/api/v1/watchlist/items` |
| `PUT` | `/api/v1/watchlist` (bulk replace / re-sync) |
| `GET` | `/api/v1/market/ticker/24h` (quote enrichment) |

OpenAPI: `backend/api/openapi/openapi.yaml`.

---

## 4. Where the code will live

| Area | Path |
|------|------|
| RTK | `mobile/src/libs/api/endpoints/watchlistApi.ts` |
| Helpers | `mobile/src/libs/utils/watchlist*.ts`, `clientId.ts` |
| Context | `mobile/src/modules/watchlist/context/` |
| Page | `mobile/src/modules/watchlist/pages/WatchlistPage/` |
| UI | `components/molecules/StarButton`, `organisms/WatchlistRow`, `WatchlistList` |
| Nav | `mobile/src/app/navigation/RootNavigator.tsx` + module `navigation.ts` |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Star a pair on Markets → open Watchlist tab → open detail → unstar
npm test
```

---

## 6. Known limitations / follow-ups

- No cross-device sync without auth.  
- Backend memory store is not durable.  
- Large lists may throttle quote enrichment.  
- Product web (`frontend/`) watchlist remains a separate backlog item (`WL-1` on board).
