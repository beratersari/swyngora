# Design: Mobile batch indicators (list RSI)

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-batch-indicators.md`  
**Feature:** `docs/features/mobile-batch-indicators.md`  
**Tasks:** `project-management/tasks/mobile/batch-indicators/MBIND-*.md`  
**Analysis:** `project-management/tasks/mobile/batch-indicators/MBIND-A.md`  
**Branch:** `feature/mobile-batch-indicators`  
**Depends on:** Markets, coin detail, favorites (done); pumps optional  
**Backend:** Existing `POST /api/v1/market/indicators/batch` — **no backend work**  
**MCP:** Single-symbol `get_indicators` exists; batch MCP tool optional follow-up  

---

## 1. Problem / goal

List screens show price but not **momentum context**. Coin detail already loads full RSI/EMA series via `GET /indicators`. Loading that per row would N+1 the API.

**Goal:** Use **batch latest snapshots** to show RSI (optional EMA) on:

1. **Favorites** (P0) — small symbol sets, high value  
2. **Markets** visible page (P1) — top of list / current page  

Follow Atomic Design + ViewModel; Chrome primary.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Batch RTK endpoint | Alerts when RSI crosses 30/70 |
| Favorites RSI badges | Paper trading / signals productization |
| Markets list RSI (controlled poll) | Full heatmap / screener page |
| Chunk ≤50, multi-exchange split | Editable periods on lists |
| Partial failure UX + disclaimer | Web frontend batch columns |
| Keep detail on GET /indicators | startTime/endTime candle ranges |

**Disclaimer:** informational only — not financial advice (API `note` or product constant).

---

## 3. Backend contract (summary)

Full matrix: **MBIND-A**.

| Method | Path | Role |
|--------|------|------|
| `POST` | `/api/v1/market/indicators/batch` | Latest RSI/EMA for ≤50 symbols, one exchange |
| `GET` | `/api/v1/market/indicators` | Full series — **detail only** (already shipped) |

### Batch defaults (mobile v1)

| Param | Value |
|-------|--------|
| exchange | From list group |
| interval | `1h` |
| rsiPeriod | `14` |
| emaPeriods | `12,26` |
| symbols | unique, ≤50 |

### Service caps

- Max **50** symbols after dedupe  
- Per-item `error: "unavailable"` on failure  
- Process-wide upstream concurrency (24) — avoid aggressive dual-screen poll storms  

---

## 4. UX flows

### Favorites (P0)

```text
Open Favorites → items ready
  → group by exchange
  → for each exchange: POST batch(symbols)
  → join items[].rsi onto rows (with ticker enrichment)
  → poll while focused + AppState active (slower than or equal to quote poll)
  → background → pause
```

### Markets (P1)

```text
Markets first page loaded
  → POST batch(exchange, symbols of rows)
  → join RSI to MarketRow
  → revalidate on pull-to-refresh; optional slow poll when focused
  → pagination: either re-batch visible set or first-page only (document choice in MBIND-5)
```

### Coin detail

Unchanged — full series + chart overlays.

---

## 5. Architecture

```text
libs/api/endpoints/marketApi.ts   (+ postIndicatorsBatch)
libs/utils/                         batchIndicators*, formatRsi
config/                             batchIndicatorsConstants (caps, defaults, poll ms)
components/molecules/rsi-badge/
components/organisms/watchlist-row | market-row  (+ optional rsi props)
modules/watchlist/.../WatchlistPage*
modules/markets/.../MarketsPage*
```

### ViewModel responsibilities

- Build batch args from pairs/rows  
- Skip empty  
- Merge snapshot map into row VMs  
- Expose batchError / isBatchLoading without blocking primary list  

---

## 6. Implementation order

| Step | Task | Notes |
|------|------|-------|
| 0 | MBIND-A | Analysis (this epic’s contract) |
| 1 | MBIND-1 | RTK only |
| 2 | MBIND-2 | Pure helpers + tests |
| 3 | MBIND-3 | Atomic badge + row props |
| 4 | MBIND-4 | Favorites wire-up |
| 5 | MBIND-5 | Markets wire-up |
| 6 | MBIND-6 | Failure/disclaimer polish |
| 7 | MBIND-7 | Docs/board/changelog |

**MR grouping:** `(A+1–2)` · `(3–4)` · `(5–6)` · `(7)`

---

## 7. Testing plan

| Layer | What |
|-------|------|
| Utils | chunk, groupByExchange, index by symbol, formatRsi, tone |
| RTK | query args / body serialization (mock baseQuery if needed) |
| Components | badge loading/value/unavailable; rows optional RSI |
| Pages | Favorites skips empty; join map; pause when unfocused (mock hooks) |

Manual: Chrome Favorites with 2–3 pairs → RSI appears; kill API → prices remain, RSI `—` / banner.

---

## 8. Risks

| Risk | Mitigation |
|------|------------|
| Batch latency (many symbols × candles) | Cap 50; Favorites first; slow poll |
| Multi-exchange favorites | Split requests |
| Spot poll + batch poll storm | Stagger intervals; pause off-screen |
| Over-interpreting RSI colors as signals | Disclaimer; neutral copy (“RSI”) |

---

## 9. Success metrics (qualitative)

- Favorites show RSI without N ticker-style indicator calls  
- No empty-list regressions when batch fails  
- Docs + board closed  
