# Feature: Multi-exchange spot markets (frontend)

**Status:** Implemented (Epic B)  
**Surface:** Product web (`frontend/`)  
**Backend:** Already implemented  
**Epic:** `frontend` · multi-exchange spot markets  
**UI kit:** Ant Design (Table, Tabs, Form controls)  
**Charts:** Not required for list phase; detail later uses Lightweight Charts  
**Local tasks:** `project-management/tasks/frontend/markets/MKT-*.md`

---

## 1. Problem / goal

Users need a production UI to browse **crypto spot markets across Binance, Coinbase, and Bybit**: search, filter, sort, paginate, and see live-ish prices and market caps.

Backend already exposes this; product frontend does not.

---

## 2. Behavior

### Happy path

1. User opens Markets (home).
2. Default: `exchange=binance`, `quote=USDT`, `sort=quoteVolume`, `order=desc`, `limit=50`.
3. Table loads from `GET /api/v1/market/spot`.
4. User switches exchange tabs → refetch with new `exchange` (reset offset).
5. User searches (`q`), picks tags, changes sort/order, paginates.
6. While tab is visible, list revalidates ~every 10s.
7. Price/change cells can flash on update (optional polish).

### Important limits

- No auth.
- Supply/mcap may be null; mcap sort can 502 if supply snapshot cold.
- Coinbase symbols differ (`BTC-USD` vs `BTCUSDT`).
- Tags primarily from Binance catalog.
- Trade count often unavailable off Binance.
- Rate limit: avoid hammering on every keystroke (debounce `q` ~300ms).

---

## 3. APIs (contract)

OpenAPI: `backend/api/openapi/openapi.yaml`

| Use | Method | Path | Notes |
|---|---|---|---|
| Exchange list | `GET` | `/api/v1/market/exchanges` | Tabs source |
| Product tags | `GET` | `/api/v1/market/tags?exchange=` | Filter chips |
| Spot markets | `GET` | `/api/v1/market/spot` | Main table |

### Spot query parameters

| Param | UI control |
|---|---|
| `exchange` | Exchange tabs |
| `q` | Search field |
| `quote` | Quote select (default USDT) |
| `base` | Optional advanced filter |
| `status` | Optional (default TRADING if useful) |
| `tag` / `tags` | Multi-select chips (OR) |
| `sort` | Column headers |
| `order` | Column headers |
| `limit` | Page size |
| `offset` | Pagination |

### Spot row fields used in UI (phase 1)

| Field | Column |
|---|---|
| `symbol` | Symbol |
| `lastPrice` | Last |
| `priceChangePercent` | 24h % |
| `quoteVolume` | Quote volume |
| `volume` | Base volume (optional column) |
| `marketCapCirculating` | Circ. mcap |
| `tags` | Tags |
| `tradeCount` | Trades (hide/— when 0 and non-Binance) |

---

## 4. Where the code lives

| Area | Path |
|---|---|
| Page | `frontend/src/components/pages/MarketsPage/` |
| Domain UI (organisms) | `frontend/src/components/organisms/` (MarketsTable, MarketsToolbar, ExchangeTabs) |
| RTK endpoints | **`frontend/src/libs/api/endpoints/marketApi.ts`** |
| Generated types | **`frontend/src/libs/api/generated/`** |
| Shared hooks | **`frontend/src/libs/hooks/`** (e.g. visibility, debounce) |
| Shared utils | **`frontend/src/libs/utils/`** (price format, exchange symbols) |
| System design | `docs/design/frontend-system-design.md` |

Atomic rules: prefixed `Name.types.ts` / `Name.constants.ts` / `Name.helpers.ts`.  
**No `src/features/`** — domain UI is organisms; REST only in `libs/api`.

---

## 5. UI components (Atomic map)

| Level | Component | Responsibility |
|---|---|---|
| Atom | Text/Spinner/Badge (antd Typography, Spin, Tag) | primitives |
| Molecule | SearchField, SelectField, TagChip (antd Input/Select/Tag) | controls |
| Organism | ExchangeTabs, MarketsToolbar, MarketsTable (antd Tabs/Table) | sections |
| Template | `MarketsTemplate` | layout chrome |
| Page | `MarketsPage` | wire RTK + state |

---

## 6. State model

### Server state (RTK Query via `@/libs/api`)

- `useListExchangesQuery()`
- `useListProductTagsQuery({ exchange })`
- `useListSpotMarketsQuery(args, { pollingInterval, skip })`

### Client/UI state

| State | Storage |
|---|---|
| `exchange`, `q`, `quote`, `tag[]`, `sort`, `order`, `limit`, `offset` | URL search params (preferred) and/or component state |
| Poll interval (5/10/15/off) | local UI state + optional localStorage |
| Column visibility (later) | localStorage |

**Prefer URL sync** so links are shareable:  
`/markets?exchange=bybit&quote=USDT&sort=marketCapCirculating&order=asc`

---

## 7. Cache / refresh

| Trigger | Behavior |
|---|---|
| Mount | Fetch spot + exchanges + tags |
| Exchange change | Invalidate/refetch tags + spot; offset=0 |
| Filter change | Debounced refetch; offset=0 |
| Poll tick | Refetch spot only if document visible |
| Window focus | Refetch if stale |
| 429 | Back off poll; toast/banner |
| 502 | Error panel with retry |

Align with backend TTLs (spot ~5s) — UI poll ≥5s, default 10s.

---

## 8. Error & empty states

| Case | UI |
|---|---|
| Loading first paint | Skeleton table |
| Empty `items` | “No markets match filters” |
| Network / 502 | Retry button |
| 429 | “Slow down” + auto-retry |
| Partial null mcap | Em dash `—` |

---

## 9. Testing / verify

```bash
# backend
cd backend && go run ./cmd/server

# frontend (after init epic)
cd frontend && npm run dev
```

Manual:

1. Switch Binance → Coinbase → Bybit; table updates.  
2. Search `btc`; results filter.  
3. Sort by quote volume and change %.  
4. Tag filter on Binance (e.g. Meme).  
5. Hide tab → poll pauses; show tab → resumes.  

Automated:

- `libs/utils`: formatters, query-arg builders  
- `libs/hooks`: visibility / debounce  
- MarketsTable: sort header callbacks, empty state  
- `libs/api`: mock baseQuery for success/error shapes  

---

## 10. Issue breakdown (GitLab)

| ID | Title | Depends |
|---|---|---|
| MKT-1 | RTK market endpoints (exchanges, tags, spot) | Epic init done |
| MKT-2 | Markets page shell + ExchangeTabs | MKT-1 |
| MKT-3 | MarketsTable + column formatters | MKT-2 |
| MKT-4 | Toolbar: search, quote, tags, sort | MKT-3 |
| MKT-5 | Pagination + URL query sync | MKT-4 |
| MKT-6 | Live poll + visibility pause | MKT-3 |
| MKT-7 | Empty/error/rate-limit UX + tests | MKT-4, MKT-6 |

---

## 11. Known limitations / follow-ups

- Detail page navigation (double-click → detail) is a **follow-up epic**.  
- Watchlist stars are a **follow-up epic**.  
- Cross-exchange volume comparison view not in phase 1.  
- `simple-frontend` remains for API harness; do not block on feature parity polish.

---

## 12. Success criteria

- [ ] User can browse live multi-exchange spot markets in product `frontend/`  
- [ ] Filters/sort/pagination work against real backend  
- [ ] Polling respects background/visibility  
- [ ] Types from OpenAPI; no hand-rolled API DTOs  
- [ ] Feature documented (this file) + README run steps  

**Last updated:** 2026-07-26
