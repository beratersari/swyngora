# Design: Mobile pump / dump radar

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-pumps.md`  
**Feature:** `docs/features/mobile-pumps.md`  
**Tasks:** `project-management/tasks/mobile/pumps/MPUMP-*.md`  
**Analysis:** `project-management/tasks/mobile/pumps/MPUMP-A.md`  
**Branch:** `feature/mobile-pumps` (from latest `develop` after watchlist merges)  
**Depends on:** Markets + coin detail (done); watchlist optional for deep-links  
**Backend:** Existing `GET /api/v1/market/pumps` + `/pumps/scan` — **no backend work**  
**MCP:** Already exposed (`detect_pump_events`, `scan_pump_events`)

---

## 1. Problem / goal

Users want to discover **rapid price moves** across top-volume pairs and inspect pump/dump events on a single coin. Backend already implements mechanical threshold detection over OHLCV.

**Goal:** Ship a mobile **Pumps** experience that:

1. Scans top quote-volume symbols for recent pumps/dumps (`/pumps/scan`)  
2. Lists ranked hits with best return % and key meta  
3. Opens coin detail; shows **per-symbol event timeline** (`/pumps`)  
4. Exposes safe defaults + advanced threshold controls  
5. Surfaces “informational only” disclaimer  
6. Follows **Atomic Design + View + ViewModel**, **kebab-case folders**, RTK under `libs/api`

Primary runtime: **Chrome** via `npm run web`.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Pumps tab (scan radar) | Price alerts / push notifications |
| Detail pumps section | Trading / paper orders |
| Filters: exchange, quote, interval, lookback, threshold, direction | AI chat interpretation of pumps |
| Manual refresh + AppState-aware (no aggressive poll) | Web `frontend/` pumps UI (separate) |
| Navigate hit → coin detail | Historical calendar picker (start/endTime) |
| Unit tests for helpers + pages | Changing backend OpenAPI schema (optional follow-up) |

**Disclaimer (must ship in UI):** copy from API `note` or product string — mechanical thresholds, not financial advice.

---

## 3. Backend contract (summary)

Full field matrix: **MPUMP-A**.

| Method | Path | Role |
|--------|------|------|
| `GET` | `/api/v1/market/pumps` | Events for one `symbol` |
| `GET` | `/api/v1/market/pumps/scan` | Ranked hits over top-volume symbols |

### Scan defaults (mobile v1)

| Param | Value |
|-------|--------|
| exchange | user selection (default binance) |
| quote | USDT (USD on Coinbase) |
| interval | `15m` |
| lookbackHours | `24` |
| minReturnPct | `8` |
| direction | `up` |
| symbolLimit | `15` |
| mode | `close_return` |

### Detail defaults

| Param | Value |
|-------|--------|
| interval | detail page interval or `1h` |
| minReturnPct | `5` |
| direction | `both` |
| maxEvents | `10` |
| lookbackHours | `48` (or limit 200) |

Scan may take several seconds (≤40 symbols × candles, concurrency 5) — design for **long loading**, not 10s poll.

---

## 4. Architecture

```text
mobile/src/
├── config/
│   └── pumpConstants.ts          # defaults, lookback options, mode labels
├── libs/
│   ├── api/
│   │   ├── baseApi.ts            # tagType: 'Pump' (add)
│   │   └── endpoints/
│   │       └── pumpApi.ts        # getPumpEvents, scanPumpEvents
│   └── utils/
│       ├── formatPump.ts         # formatReturnPct, tone, volume ratio
│       └── pumpQuery.ts          # build scan/detail query args
├── modules/
│   └── pumps/                    # NEW module
│       ├── navigation.ts
│       ├── pumps.types.ts
│       └── pages/
│           └── pumps-scan-page/  # View + ViewModel (kebab folder)
└── components/
    ├── molecules/
    │   └── pump-return-badge/    # +12.4% / −8.1% tone
    └── organisms/
        ├── pump-scan-filters/    # exchange, lookback, threshold, direction
        ├── pump-hit-row/         # symbol, bestReturn, event count
        ├── pump-hit-list/        # FlatList
        └── pump-event-list/      # detail timeline rows
```

**Import style:** `@/components/organisms/pump-hit-row`  
**No** `modules/*/components/`.

### Navigation

```text
Main tabs:
  HomeTab
  MarketsTab
  FavoritesTab   (existing, conditional)
  PumpsTab       # NEW — always visible (discovery surface)

Pumps stack:
  PumpsScan      # list
  CoinDetail     # reuse CoinDetailPage with exchange+symbol
                 # (or navigate to MarketsTab detail — prefer stack-local Detail for parity with favorites)
```

**Recommendation:** Mirror favorites — `CoinDetail` on Pumps stack reusing `CoinDetailPage`.

Detail pumps section: organism embedded in `CoinDetailPage` when data loads (same page, no new route).

---

## 5. UX (phone-width)

### 5.1 Pumps tab — scan

```text
┌─────────────────────────────────────┐
│ Pumps                               │
├─────────────────────────────────────┤
│ [Binance] [Coinbase] [Bybit]        │
│ Quote USDT · 15m · 24h · ≥8% · Up   │  ← chips / open filters sheet
├─────────────────────────────────────┤
│ BTCUSDT     +18.4%   3 events       │
│             15m · vol ×2.1          │
│ SOLUSDT     +12.1%   1 event        │
│ …                                   │
├─────────────────────────────────────┤
│ Informational only — not advice     │
└─────────────────────────────────────┘
```

| Interaction | Behavior |
|-------------|----------|
| Pull-to-refresh | Refetch scan |
| Row press | Coin detail |
| Change exchange / filters | Reset + refetch |
| Empty | “No pumps matched — try lower threshold or longer lookback” |
| Loading | Skeleton list (scan is slow) |
| App backgrounded | Do not poll; optional cancel in-flight |

### 5.2 Coin detail — events section

Below indicators (or after stats):

```text
│ Pump / dump events                  │
│ ≥5% · both · 1h · last 48h          │
│ +9.2%  12:00  close_return  vol×1.4 │
│ −6.1%  08:00  …                     │
│ (disclaimer)                        │
```

Optional: “Open full pumps scan” deep-link to Pumps tab with same exchange.

---

## 6. RTK surface (`pumpApi.ts`)

```text
scanPumpEvents   query  GET /api/v1/market/pumps/scan
getPumpEvents    query  GET /api/v1/market/pumps
```

- Types: hand-map from MPUMP-A until OpenAPI schemas are tightened; prefer `operations` if generated includes params  
- Tags: `{ type: 'Pump', id: scanKey }` / `{ type: 'Pump', id: \`${ex}:${sym}:...\` }`  
- `keepUnusedDataFor`: 30–60s  
- **No** `pollingInterval` for scan in v1  

Add `'Pump'` to `baseApi` tagTypes.

---

## 7. Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module | `modules/pumps` | Separate discovery surface from markets list |
| Tab visibility | Always show Pumps tab | Discovery; unlike favorites empty-hide |
| Scan polling | Manual / pull only | Expensive multi-symbol work |
| Defaults | Match service (15m / 24h / 8%) | Fewer surprise empty results |
| Mode UI | Advanced collapsed | Most users only need threshold + direction |
| Detail section | On CoinDetailPage | Context without new route |
| Folder naming | kebab-case | Mobile AGENTS convention |
| Backend | No changes | API + MCP ready |
| OpenAPI looseness | Client types from MPUMP-A | Optional backend follow-up |

---

## 8. Task breakdown (MPUMP)

| ID | Title |
|----|-------|
| MPUMP-A | API field matrix analysis (this folder) |
| MPUMP-1 | RTK `pumpApi` + baseApi Pump tag |
| MPUMP-2 | `formatPump` + `pumpQuery` helpers + tests |
| MPUMP-3 | Organisms: filters, hit row/list, return badge |
| MPUMP-4 | PumpsScanPage View + ViewModel + tab navigation |
| MPUMP-5 | Coin detail pump events section |
| MPUMP-6 | Loading / empty / error / disclaimer + AppState |
| MPUMP-7 | Tests, board, feature/design status, changelog |

**MR grouping:** (A+1–2) · (3–4) · (5–6) · (7)

Branch: `feature/mobile-pumps` → MR to `develop`.

---

## 9. Testing plan

| Layer | What |
|-------|------|
| Unit | formatReturnPct, direction tone, buildScanQuery, buildDetailQuery |
| Unit | ViewModel query-arg changes when filters change |
| Component | PumpHitRow, PumpEventList, filters chips |
| Page | Empty / error / loading with injected viewModel |
| Manual | Chrome: scan Binance USDT → open hit → detail events; lower threshold |

```bash
cd mobile && npm test && npm run lint && npm run typecheck
cd mobile && npm run web   # backend :8080
```

---

## 10. Risks

| Risk | Mitigation |
|------|------------|
| Slow scan / timeouts | Skeleton; symbolLimit 15 default; surface error + retry |
| Empty results | Sensible defaults; empty copy suggests lower threshold |
| User confuses with trading signals | Persistent disclaimer from API note |
| Rate limits | No poll; debounce filter changes 300ms |
| OpenAPI incomplete types | MPUMP-A as contract; optional OpenAPI tighten later |

---

## 11. Definition of done

- [ ] Pumps tab lists scan hits for selected exchange  
- [ ] Filters change query and refetch  
- [ ] Hit opens coin detail  
- [ ] Detail shows pump events for symbol  
- [ ] Disclaimer visible  
- [ ] Atomic kebab folders; ViewModels without JSX  
- [ ] Tests green; docs/board updated  

**Last updated:** 2026-07-27
