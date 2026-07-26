# Design: Mobile coin detail + indicators

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-coin-detail.md`  
**Tasks:** `project-management/tasks/mobile/detail/MDET-*.md`  
**Branch:** `feature/mobile-coin-detail`  
**Depends on:** Mobile markets dashboard (MMKT)  
**Web parity:** `docs/features/coin-detail.md` + `tasks/frontend/detail/DET-A.md`, `DET-B.md`

---

## 1. Goal

Ship a mobile **CoinDetailPage** opened from Markets list rows:

1. Route params: `exchange`, `symbol`; optional `interval` / `limit`  
2. Ticker + supply stats  
3. Interval control  
4. OHLCV chart  
5. RSI pane + EMA overlays  
6. Atomic UI only; View + ViewModel; RTK under `libs/api`

---

## 2. Navigation

| From | To |
|------|-----|
| `MarketsList` row press | `CoinDetail` with `{ exchange, symbol }` |
| Detail back | Markets stack pop |

```text
Markets stack:
  MarketsList
  MarketsFilters
  CoinDetail   ← new
```

Replace “detail coming soon” toast with `navigation.navigate(MarketsScreens.Detail, { exchange, symbol })`.

---

## 3. Architecture

```text
modules/markets/   # or modules/detail/ if preferred later
  pages/CoinDetailPage/
    CoinDetailPage.tsx
    CoinDetailPage.viewModel.ts
    CoinDetailPage.types.ts
    …
  navigation.ts     # add CoinDetail params

components/ (Atomic only)
  organisms/
    CoinDetailHeader/
    CoinDetailStats/
    IntervalToolbar/
    CandleChart/          # host for RN-web chart
    IndicatorRsiPane/
    …
  molecules/
    StatTile/
    IntervalChip/         # or reuse Chip/ChipGroup
```

**No** `modules/*/components/`.

### ViewModel responsibilities

- Read route params  
- RTK: intervals, ticker, supply, candles, indicators  
- Local interval/limit state  
- AppState + focus → polling intervals (ticker ~15s, candles ~30s; pause when inactive)  
- Map API → presentational props for organisms  

---

## 4. Chart strategy (mobile / Chrome)

Web product uses **TradingView Lightweight Charts** (DOM). On **react-native-web**, evaluate in MDET-4:

| Option | Note |
|--------|------|
| A. Lightweight Charts in a web-only host | Prefer if DOM available under RN-web |
| B. SVG/path simple candle fallback | Minimal if A blocked |
| C. Defer full chart | Ship stats + indicators tables first (not preferred) |

**Decision default:** Option A with a thin `CandleChart` organism; document limitation in package README. No Expo chart kits.

---

## 5. Defaults

| Param | Default |
|-------|---------|
| interval | `1h` |
| candle limit | `100` |
| RSI period | `14` |
| EMA periods | `12`, `26` |

---

## 6. Task breakdown (MDET)

| ID | Title |
|----|-------|
| MDET-1 | Extend marketApi for detail queries (intervals, ticker, supply, candles, indicators) |
| MDET-2 | Navigation: CoinDetail route + row press wiring |
| MDET-3 | CoinDetailPage shell + header/stats organisms + ViewModel |
| MDET-4 | Interval toolbar + candle chart organism |
| MDET-5 | RSI/EMA indicators organisms + series mapping |
| MDET-6 | Section loading/error, polling pause, tests |
| MDET-7 | Docs, board, changelog closeout |

**MR grouping:** (1–2) · (3) · (4–5) · (6–7)

---

## 7. Definition of done

- [x] Detail opens from markets row with correct exchange/symbol  
- [x] Stats + chart + indicators visible on Chrome  
- [x] Atomic-only UI; ViewModel has no JSX  
- [x] Polling respects focus/AppState  
- [x] Tests green; design/feature docs marked implemented when shipped  

**Last updated:** 2026-07-26
