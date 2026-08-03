# Feature: Home dashboard (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing spot, pumps scan, watchlist, health  
**Epic:** `project-management/epics/mobile-home-dashboard.md`  
**Design:** `docs/design/mobile-home-dashboard.md`  
**Tasks:** `project-management/tasks/mobile/home/MHOME-*.md`

---

## 1. Problem / goal

Give mobile users a **landing dashboard** with live market context instead of API health alone.

---

## 2. Behavior (happy path)

1. Open **Home** tab.  
2. See favorites (if any), top movers, high volume, pump teaser.  
3. Tap a row → coin detail.  
4. Pull to refresh.  
5. Quick chips → Markets / Pumps / Ask.  

### Limits

- Read-only widgets; no trading.  
- Default Binance / USDT for list widgets.  
- Informational disclaimers on pump/RSI surfaces.

---

## 3. APIs

| Method | Path |
|--------|------|
| `GET` | `/api/v1/market/spot` |
| `GET` | `/api/v1/market/pumps/scan` |
| `GET` | `/api/v1/market/ticker/24h` (favorites quotes) |
| `POST` | `/api/v1/market/indicators/batch` (optional RSI on favorites) |
| `GET` | `/health` (optional footer) |

---

## 4. Code homes (planned)

| Area | Path |
|------|------|
| Page | `mobile/src/modules/app/pages/home-page/` |
| Organisms | `mobile/src/components/organisms/dashboard-*` etc. |
| Helpers | `mobile/src/libs/utils/homeDashboardQuery.ts` |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Home tab shows live widgets
```

---

## 6. Known limitations

- Not a customizable widget board.  
- Single default exchange for movers/volume until settings exist.  
