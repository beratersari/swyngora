# Design: Mobile home dashboard

**Status:** Planned  
**Epic:** `project-management/epics/mobile-home-dashboard.md`  
**Feature:** `docs/features/mobile-home-dashboard.md`  
**Tasks:** `project-management/tasks/mobile/home/MHOME-*.md`  
**Branch:** `feature/mobile-home-dashboard`  
**Depends on:** Markets, detail, favorites, pumps, batch RSI, Ask (done)  
**Backend:** Existing market / watchlist / pump APIs only  

---

## 1. Problem / goal

Home is still a scaffold (health + shortcuts). Users land on an empty shell while Markets / Pumps / Favorites already have live data.

**Goal:** Open the app to a **dashboard** of market context and one-tap navigation.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Live widgets on Home | Alert rules engine |
| Reuse spot / pumps / watchlist | New OpenAPI endpoints |
| Deep links to existing tabs | Cross-exchange compare |
| Partial failure UX | Drag-and-drop layout |
| AppState poll pause | Web home redesign |

---

## 3. Data sources

| Widget | Endpoint | Defaults |
|--------|----------|----------|
| Movers | `GET /spot` | `exchange=binance`, `quote=USDT`, `sort=priceChangePercent`, `order=desc`, `limit=5` |
| Volume | `GET /spot` | `sort=quoteVolume`, `order=desc`, `limit=5` |
| Pumps | `GET /pumps/scan` | `symbolLimit=3`, existing scan defaults |
| Favorites | Watchlist context + ticker/batch | Top 5 pairs |
| Health | `GET /health` | Optional footer only |

Polling: ~15–30s while Home focused + app active (tune in constants).

---

## 4. UX flow

```text
Open Home
  → parallel fetch widgets
  → render sections as data arrives
  → pull-to-refresh → refetch all
  → tap row → Coin detail
  → See all Markets / Pumps → tab navigate
  → Ask chip → Ask tab
```

---

## 5. UI structure

```text
Home
  [Quick actions: Markets | Pumps | Ask]
  Favorites strip (or empty CTA)
  Top movers (24h %)
  Highest volume
  Pump radar teaser
  Language switcher
  API health (collapsed / caption)
```

Atomic: `section-header`, `dashboard-market-row`, `dashboard-section-list`, `pump-teaser-card`, `quick-action-chips`.

---

## 6. ViewModel sketch

```ts
type HomePageViewModel = {
  favorites: DashboardRow[];
  movers: DashboardRow[];
  volume: DashboardRow[];
  pumps: PumpTeaser[];
  // per-section loading / error
  onRefresh: () => void;
  onOpenMarkets: () => void;
  onOpenPumps: () => void;
  onOpenAsk: () => void;
  onPressMarket: (exchange: string, symbol: string) => void;
  // ...
};
```

---

## 7. Testing

Helpers unit tests; page tests with injected VM; no live network in unit tests.

---

## 8. Task map

| ID | Focus |
|----|--------|
| MHOME-1 | Helpers + constants |
| MHOME-2 | Atomic UI |
| MHOME-3 | ViewModel |
| MHOME-4 | View + navigation |
| MHOME-5 | Error / empty polish |
| MHOME-6 | Tests |
| MHOME-7 | Docs closeout |
