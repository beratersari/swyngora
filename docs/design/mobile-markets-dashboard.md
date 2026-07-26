# Design: Mobile markets dashboard (multi-exchange spot)

**Status:** Implemented  
**Epic:** `mobile` · `epic::mobile-markets`  
**Priority:** P0 mobile track (after init MR)  
**Branch:** `feature/mobile-spot-markets`  
**Depends on:** `feature/mobile-init` (scaffold)  
**Parity reference:** `docs/features/multi-exchange-spot-markets.md` (web)  
**Local PM:** `project-management/epics/mobile-multi-exchange-spot-markets.md`

---

## 1. Problem / goal

Mobile init ships a **Markets stub**. Users need a real **markets dashboard**: the primary surface to scan multi-exchange spot markets on a phone-width Chrome viewport (react-native-web), without Expo or Ant Design.

**Goal:** Design and implement a mobile-first dashboard under `modules/markets` that:

1. Lists spot markets for Binance / Coinbase / Bybit  
2. Supports search, quote, tags, sort, pagination  
3. Live-polls with AppState / focus pause  
4. Keeps **View + ViewModel** and Atomic boundaries  
5. Reuses **frontend brand tokens** and the same OpenAPI contract  

---

## 2. Product framing: “Dashboard”

On mobile, the **Markets tab is the market data dashboard** (not a separate analytics home). Scope of this epic:

| In dashboard epic | Out of scope (later) |
|-------------------|----------------------|
| Exchange-scoped spot list | Coin detail / candles / RSI |
| Filters + sort + pagination | Watchlist stars |
| Live poll + pull-to-refresh | AI chat, alerts |
| Empty / error / loading UX | Native-only gestures beyond RN Web |

Optional polish (same epic if time): compact **summary strip** (active exchange, row count, last updated).

---

## 3. Goals & non-goals

### Goals

1. Replace MarketsPage stub with full dashboard View + ViewModel.  
2. RTK endpoints: `listExchanges`, `listProductTags`, `listSpotMarkets` in `libs/api/endpoints/marketApi.ts`.  
3. Module components: exchange chips, filter bar, market row, list.  
4. Defaults aligned with web: `binance`, `USDT`, `quoteVolume` desc.  
5. Mobile page size default **30** (vs web 50) for list performance.  
6. Chrome via `npm run web`; no simulator required.  
7. Tests for formatters, VM query-arg building, empty/error states.

### Non-goals

- Port Ant Design Table 1:1  
- Charts / detail route (backlog `MDET-*`)  
- Auth  
- Offline-first cache beyond RTK defaults  
- Shared monorepo package for types  

---

## 4. UX design (phone-width)

```text
┌─────────────────────────────────────┐
│ Markets                    (title)  │
├─────────────────────────────────────┤
│ [Binance] [Coinbase] [Bybit]        │  ← horizontal chips / tabs
├─────────────────────────────────────┤
│ 🔍 Search pairs…                    │
│ Quote [USDT ▾]   Tags [+ filter]    │
│ Sort  [Quote vol ▾]  [Desc ▾]       │
├─────────────────────────────────────┤
│ BTCUSDT    $67,123.4    +1.24%      │
│            Vol 1.2B    Mcap 1.3T    │  ← MarketRow
│ ETHUSDT    …                        │
│ …                                   │
├─────────────────────────────────────┤
│ ‹ Prev    1–30 of 412    Next ›     │
└─────────────────────────────────────┘
```

### Interaction notes

| Control | Behavior |
|---------|----------|
| Exchange chip | Sets `exchange`, resets `offset`, clears tags if exchange-specific |
| Search | Debounce 300ms → `q` |
| Quote | Select; default USDT (USD on Coinbase if product later needs it — start USDT, document Coinbase `USD` if API expects) |
| Tags | Multi-select modal/sheet; OR semantics (`tags=a,b`) |
| Sort | Picker of API sort fields |
| Row press | No-op or “Detail soon” toast in this epic |
| Pull-to-refresh | `refetch()` on spot (+ optional tags) |
| Poll | 10s when `useAppStateActive() && useIsFocused()` |

### Visual

- Canvas navy, cards muted, cream primary text, steel secondary (existing tokens).  
- 24h %: success green / error red from `semanticColors.status`.  
- Skeleton rows while first load.  
- Sticky filter block optional; prefer scroll-all for web simplicity in v1.

---

## 5. Architecture

```text
modules/markets/
  pages/MarketsPage/
    MarketsPage.tsx              # View only
    MarketsPage.viewModel.ts     # RTK + debounce + poll + nav state
    MarketsPage.types.ts         # MarketsPageViewModel (normative)
    MarketsPage.styles.ts
    MarketsPage.constants.ts
    MarketsPage.helpers.ts       # map SpotMarket → row VM if needed
    MarketsPage.test.tsx
    MarketsPage.viewModel.test.ts
  components/                    # flat module UI
    ExchangeChips/
    MarketsFilterBar/
    MarketRow/
    MarketsList/
    MarketsPagination/
  navigation.ts
```

### Normative ViewModel (target)

```ts
export type MarketRowViewModel = {
  id: string; // symbol
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  quoteVolumeLabel: string;
  marketCapLabel: string;
  tagsLabel: string;
};

export type MarketsPageViewModel = {
  title: string;
  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;

  search: string;
  onSearchChange: (q: string) => void;

  quote: string;
  quoteOptions: string[];
  onQuoteChange: (quote: string) => void;

  availableTags: string[];
  selectedTags: string[];
  onToggleTag: (tag: string) => void;
  onClearTags: () => void;

  sort: string;
  order: 'asc' | 'desc';
  sortOptions: { value: string; label: string }[];
  onSortChange: (sort: string) => void;
  onOrderChange: (order: 'asc' | 'desc') => void;

  rows: MarketRowViewModel[];
  total: number;
  offset: number;
  limit: number;
  onNextPage: () => void;
  onPrevPage: () => void;
  canNext: boolean;
  canPrev: boolean;

  isLoading: boolean;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  lastUpdatedLabel: string | null;

  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (symbol: string) => void; // stub ok
};
```

### Data layer

Port web `marketApi` list endpoints (exchanges, tags, spot) into `mobile/src/libs/api/endpoints/marketApi.ts` using generated OpenAPI types. Wire store import like health.

### Formatters

Port pure helpers from frontend (`formatPrice`, `formatChangePercent`, `formatCompactUsd`, `changeTone`) into `mobile/src/libs/utils/` — **no React, no antd**.

### State ownership

| State | Where |
|-------|--------|
| exchange, q, quote, tags, sort, order, offset | ViewModel local state (useState) |
| Server data | RTK Query |
| Debounced q | `useDebouncedValue` in libs/hooks |
| Focused screen | `@react-navigation` `useIsFocused` |
| App active | `useAppStateActive` |

**No URL query sync required for v1** (web has URL sync; mobile uses in-memory VM state). Optional later: deep link `?exchange=`.

---

## 6. Defaults & constants

```ts
// MarketsPage.constants.ts
export const DEFAULT_EXCHANGE = 'binance';
export const DEFAULT_QUOTE = 'USDT';
export const DEFAULT_SORT = 'quoteVolume';
export const DEFAULT_ORDER = 'desc';
export const DEFAULT_LIMIT = 30;
export const SPOT_POLL_MS = 10_000;
export const SEARCH_DEBOUNCE_MS = 300;
```

Quote options v1: `['USDT', 'USD', 'BTC', 'EUR']` (filter empty results gracefully).

---

## 7. API mapping

| UI | Query param |
|----|-------------|
| Exchange chips | `exchange` |
| Search | `q` |
| Quote | `quote` |
| Tags | `tags` comma-joined OR |
| Sort / order | `sort`, `order` |
| Page | `limit`, `offset` |

Handle 502 on cold mcap sort: show `errorMessage` + Retry (same as web).

---

## 8. Loading / empty / error

| State | UI |
|-------|-----|
| First load | Skeleton rows (3–6) |
| Refresh | Pull indicator / subtle isRefreshing (keep prior rows) |
| Empty (200, 0 rows) | “No markets match filters” |
| Error | Message + Retry button |
| Polling paused | Optional caption (debug/dev only; product can hide) |

---

## 9. Testing plan

| Test | Coverage |
|------|----------|
| `formatMarket.test.ts` | compact USD, change %, tone |
| `MarketsPage.viewModel.test.ts` | query args, debounce, page bounds (mock RTK) |
| `MarketsPage.test.tsx` | empty / error / rows via injected VM |
| `MarketRow.test.tsx` | renders labels |

---

## 10. Issue breakdown (MMKT)

| ID | Title | Est. |
|----|-------|------|
| MMKT-1 | RTK marketApi: exchanges, tags, spot | M |
| MMKT-2 | Formatters + spot query helpers in libs/utils | S |
| MMKT-3 | ExchangeChips + MarketsFilterBar module components | M |
| MMKT-4 | MarketRow + MarketsList | M |
| MMKT-5 | MarketsPage ViewModel (filters, poll, pagination) | L |
| MMKT-6 | MarketsPage View wiring + pull-to-refresh | M |
| MMKT-7 | Empty/error/loading UX + tests | M |
| MMKT-8 | Docs: feature note, README, board; dual-codegen note | S |

**MR grouping**

| MR | Tasks |
|----|-------|
| MR1 | MMKT-1 + MMKT-2 |
| MR2 | MMKT-3 + MMKT-4 |
| MR3 | MMKT-5 + MMKT-6 |
| MR4 | MMKT-7 + MMKT-8 |

---

## 11. Risks

| Risk | Mitigation |
|------|------------|
| Large lists on web RN | limit 30; virtualize later if needed |
| Coinbase quote USD vs USDT | document; try USDT first; add USD default when exchange=coinbase in VM |
| Init MR not merged | branch forked from `feature/mobile-init`; rebase onto develop after !15 merges |
| Tag overload on small screens | modal multi-select, not infinite chips |

---

## 12. Definition of done

- [x] Markets tab shows live spot data for at least Binance  
- [x] Exchange / search / quote / tags / sort / pagination work  
- [x] Polling pauses when document hidden or app inactive  
- [x] View has zero RTK imports  
- [x] `npm test`, `typecheck`, `lint` green  
- [x] `npm run web` demo works in Chrome with backend up  
- [x] Design + tasks + board updated  

**Last updated:** 2026-07-26
