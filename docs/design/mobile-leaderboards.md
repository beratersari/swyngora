# Design: Mobile gainers / losers / high-volume leaderboards

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-leaderboards.md`  
**Feature:** `docs/features/mobile-leaderboards.md`  
**Tasks:** `project-management/tasks/mobile/leaderboards/MLEAD-*.md`  
**Branch:** `feature/mobile-leaderboards`  
**Depends on:** Markets + Home dashboard (done)  
**Backend:** Existing `GET /api/v1/market/spot` only — **no new endpoints**

---

## 1. Problem / goal

Home already shows **top movers** and **highest volume** teasers (limit 5). Users cannot:

- See a **full ranked list** (pagination / longer top N)
- Browse **losers** (24h % ascending)
- Switch **exchange / quote** for those boards without the full Markets filter UX

**Goal:** First-class **leaderboards** for gainers, losers, and high volume, reusable from Home “See all” and Markets entry.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Gainers / losers / volume boards | New OpenAPI paths |
| Full-page lists with poll + pull-to-refresh | Trade count board (optional follow-up) |
| Exchange + quote chips on board | Alerts on rank changes |
| Deep link from Home widgets | Drag-and-drop Home layout |
| Optional batch RSI on board rows (reuse MBIND) | Web (`frontend/`) redesign |

---

## 3. Data sources

| Board | Endpoint | Defaults |
|-------|----------|----------|
| Gainers | `GET /spot` | `sort=priceChangePercent`, `order=desc`, `status=TRADING` |
| Losers | `GET /spot` | `sort=priceChangePercent`, `order=asc`, `status=TRADING` |
| High volume | `GET /spot` | `sort=quoteVolume`, `order=desc`, `status=TRADING` |

Shared filters:

| Param | Default | Notes |
|-------|---------|-------|
| `exchange` | `binance` | Chip switcher (binance / coinbase / bybit) |
| `quote` | USDT (USD on coinbase default if preferred) | Chip: USDT / USD / USDC |
| `limit` | 30 (page) | Align with Markets `DEFAULT_LIMIT` |
| `offset` | 0 + infinite scroll | Reuse Markets list pagination pattern |

**Reuse:** `toSpotListQuery` / home `buildMoversSpotQuery` / `buildVolumeSpotQuery` — extend with losers builder and board-specific limits.

Optional: `POST /indicators/batch` for RSI badges on visible rows (same as Markets list; P1 within epic or skip if time-box).

---

## 4. UX flow

```text
Home → "See all" on Top movers
  → Leaderboards stack: Gainers tab (or board=gainers)
  → switch Losers | Volume chips
  → change exchange / quote
  → scroll for more
  → tap row → Coin detail

Markets → entry "Leaderboards" (toolbar or home-style link)
  → same boards
```

### Screens

**Preferred v1 (simpler):** One page `LeaderboardsPage` with:

1. Segmented control: **Gainers | Losers | Volume**  
2. Exchange chips + quote chips  
3. Ranked list (reuse `MarketRow` / `MarketsList` patterns)  
4. Rank number column or prefix `#1`…  

**Navigation:**

| Route | Stack | Params |
|-------|-------|--------|
| `Leaderboards` | Home stack and/or Markets stack | `{ board?: 'gainers' \| 'losers' \| 'volume' }` |

Deep link from Home movers → `board=gainers`; volume teaser → `board=volume`.  
Optional Home losers teaser later (out of scope unless cheap).

---

## 5. UI structure

```text
LeaderboardsPage
  [Gainers | Losers | Volume]
  [ExchangeChips]
  [Quote chips]
  MarketsList / LeaderboardList
    row: #rank · symbol · price · 24h% · vol
  empty / error / pull-to-refresh
```

Atomic:

| Component | Level | Role |
|-----------|-------|------|
| Reuse `exchange-chips`, `chip-group`, `market-row`, `markets-list` | existing | Prefer reuse |
| optional `leaderboard-segment` | molecule | Board type chips |
| optional `leaderboard-row` | organism | Rank + market metrics if MarketRow insufficient |

**No** `modules/*/components` — all UI under `components/`.

---

## 6. ViewModel sketch

```ts
type LeaderboardKind = 'gainers' | 'losers' | 'volume';

{
  board: LeaderboardKind;
  onSelectBoard(board: LeaderboardKind): void;
  exchange: MarketExchange;
  onSelectExchange(ex: string): void;
  quote: string;
  onSelectQuote(q: string): void;
  rows: MarketRowViewModel[]; // with rankLabel?
  isLoading / isLoadingMore / hasMore / error / empty
  onLoadMore / onRefresh / onPressRow / onRetry
  isPollingPaused
}
```

Polling: ~10–15s on first page when focused + AppState active (match Markets/Home hygiene).

---

## 7. Relation to Home

| Home widget | Leaderboard deep link |
|-------------|------------------------|
| Top movers “See all” | Gainers board |
| Highest volume “See all” | Volume board |
| (no losers teaser v1) | — |

Refactor Home query builders to share `libs/utils/leaderboardQuery.ts` (or extend `homeDashboardQuery`) so sort args stay consistent.

---

## 8. i18n

Namespace: prefer **`home`** for deep-link labels + new **`leaderboards`** namespace (en/tr) for page chrome.  
Keys: title, boards.gainers/losers/volume, empty, error, quote, rank a11y.

---

## 9. Testing

- Pure: `buildGainersQuery`, `buildLosersQuery`, `buildVolumeQuery`, rank index helpers  
- Page: board switch refetches; empty/error; press row  
- No live network  

---

## 10. Acceptance (epic-level)

- [ ] Full gainers / losers / volume lists (not only Home limit 5)  
- [ ] Exchange + quote filters  
- [ ] Pagination or infinite scroll  
- [ ] Home “See all” opens correct board  
- [ ] Poll pause on background / unfocus  
- [ ] en/tr i18n  
- [ ] Tests + docs closed  

## 11. Out of scope

- New backend aggregation endpoints  
- Multi-timeframe leaderboards (1h/7d) without API support  
- Custom watchlist-only boards  
- Trade-count leaderboard (unless free via `sort=tradeCount`)  
