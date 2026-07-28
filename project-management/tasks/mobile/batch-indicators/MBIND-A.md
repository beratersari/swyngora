# MBIND-A: Batch indicators API field matrix (analysis)

| Field | Value |
|---|---|
| **ID** | MBIND-A |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/batch-indicators/MBIND-A.md` |

## Purpose

Map backend **batch indicators** HTTP contract to mobile list-enrichment needs. **No backend work required** for v1 — endpoint already exists.

Sources:

- OpenAPI: `backend/api/openapi/openapi.yaml` (`postIndicatorsBatch`)
- Handler: `backend/internal/transport/http/handler/indicators.go` (`PostIndicatorsBatch`)
- Service: `backend/internal/service/market/indicators_batch.go`
- Domain defaults: RSI/EMA periods (`domain.DefaultRSIPeriod`, EMA validation)
- Single-symbol (already on mobile detail): `GET /api/v1/market/indicators`
- Product gap list: investigation vs mobile RTK surface

---

## 1. Endpoints

| Method | Path | operationId | Mobile use |
|--------|------|-------------|------------|
| `POST` | `/api/v1/market/indicators/batch` | `postIndicatorsBatch` | **List enrichment** — Favorites rows, Markets visible page (latest RSI/EMA only) |
| `GET` | `/api/v1/market/indicators` | `getIndicators` | Already used on **Coin detail** (full series + chart). **Do not replace** with batch. |

Batch is **read-only**, latest snapshot only (no `points[]`). Responses include a **note**: informational only — not financial advice.

### Why batch (vs N× GET /indicators)

| Approach | Cost | UX |
|----------|------|-----|
| N parallel `GET /indicators` | N HTTP + N series payloads | Slow, rate-limit risk |
| One `POST /indicators/batch` | 1 HTTP, latest-only, capped | List columns without N+1 |

---

## 2. Request — `POST /api/v1/market/indicators/batch`

### Body fields

| Field | Required | Default (service / OpenAPI) | Type | UI / client |
|-------|----------|-----------------------------|------|-------------|
| `symbols` | **yes** | — | `string[]` | From Favorites pairs or Markets page rows |
| `exchange` | no | `binance` | enum binance\|coinbase\|bybit | **One exchange per request** — split multi-exchange favorites by venue |
| `interval` | no | `1h` | string (venue-valid) | Default `1h`; optional chip later |
| `rsiPeriod` | no | **14** | int 2–500 | Fixed v1; align with detail (`DEFAULT_RSI_PERIOD`) |
| `emaPeriods` | no | `"12,26"` | CSV string | Fixed v1; align with detail (`DEFAULT_EMA_PERIODS`) |

### Service behavior (critical for UI)

| Rule | Value | Mobile implication |
|------|-------|-------------------|
| Max symbols after dedupe | **50** (`maxBatch`) | Chunk favorites/markets into ≤50 per request |
| Input scan cap | 500 raw entries | Client should still send ≤50 |
| Symbol normalize | per exchange | Send venue symbols (`BTCUSDT` / Coinbase style) |
| Dedupe | by normalized symbol | Safe to pass duplicates; prefer unique client-side |
| Invalid interval | 400 | Validate via `listIntervals` when user can change interval |
| Empty symbols | 400 | Skip query when list empty |
| Handler rejects | `len(symbols) > 500` → 400 | Always send ≤50 |
| Per-request concurrency | 8 | Latency grows with symbol count |
| Process-wide upstream sem | 24 | Heavy Markets poll + Favorites poll should not both thrash |
| Candle warm-up | ~max(period)+20, clamp 60–200 | Not exposed; batch is “latest only” |
| Cancel | partial items may have errors | Map `error` per symbol; do not fail whole list UI |

### Multi-exchange note

Batch is **single-exchange**. Favorites can mix binance + coinbase + bybit → **group by exchange**, one POST per group (parallel RTK queries OK if capped).

---

## 3. Response

| Field | Type | UI mapping |
|-------|------|------------|
| `exchange` | string | Meta / cache key |
| `interval` | string | Subtitle / cache key |
| `items[]` | array | Join to list rows by `symbol` |
| `note` | string | Optional footer disclaimer (once per screen, not per row) |

### `items[]` item

| Field | Type | UI |
|-------|------|-----|
| `symbol` | string | Join key (normalize case) |
| `rsi` | number \| null | **Primary** list badge (e.g. `RSI 62.4`) |
| `ema` | `Record<string, number>` keys like `"12"`, `"26"` | Secondary optional (EMA12/EMA26) or omit in v1 row |
| `error` | string | Present when failed — handler redacts to **`"unavailable"`** |

Handler never exposes raw upstream errors on items.

### Join strategy

```text
rowKey = `${exchange}|${symbol.toUpperCase()}`  // or watchKey helper
map[symbol] = item
// missing symbol → treat as pending/unavailable, keep price UI intact
```

Partial success is normal: show RSI where present; show `—` or skeleton where `error` or missing.

---

## 4. Mobile placement decisions (v1)

| Surface | Priority | Symbols source | Poll |
|---------|----------|----------------|------|
| **Favorites (Watchlist)** | **P0** | `watchlist.items` (cap enrich, e.g. 50) | AppState + focus; same order of magnitude as quote poll or slower |
| **Markets list (visible page)** | **P1** | Current page rows (≤ `DEFAULT_LIMIT` 30) | When focused; **pause** when offset > 0 pagination optional, or only first page |
| Coin detail | **Out** | Already full `getIndicators` series | Unchanged |

### v1 UX scope

| In | Out |
|----|-----|
| RSI label/badge on Favorites rows | User-editable RSI/EMA periods |
| Optional EMA secondary line or omit | Full indicator series on list |
| Interval fixed `1h` (or single chip set later) | Heatmap / screener page |
| Disclaimer once on screen | Alerts on RSI thresholds |
| Chunking if >50 favorites | Web `frontend/` batch columns |

---

## 5. RTK design sketch

```text
marketApi (or indicatorsApi inject):
  postIndicatorsBatch: build.mutation | build.query
```

**Prefer `build.query`** with body via custom `query` + `serializeQueryArgs` so cache keys work:

- Cache key: `exchange + interval + sorted(symbols).join(',') + rsiPeriod + emaPeriods`
- `providesTags`: `[{ type: 'Indicator', id: 'BATCH' }, ... per exchange]`
- Or `providesTags: ['IndicatorBatch']` + invalidate rarely (read-only market data)

**Polling:** `pollingInterval` only when AppState active + screen focused (mirror ticker / spot).

**Skip:** empty symbol list; backgrounded app.

**Chunking helper:** `chunkSymbols(symbols, 50)` → multiple queries or sequential merge (prefer one request when ≤50).

---

## 6. Atomic / ViewModel sketch

| Layer | Responsibility |
|-------|----------------|
| `libs/api` | `postIndicatorsBatch` / `usePostIndicatorsBatchQuery` |
| `libs/utils` | `buildBatchIndicatorsBody`, `chunkSymbols`, `indexBatchItemsBySymbol`, `formatRsi`, `rsiTone` |
| Atoms/molecules | e.g. `rsi-badge` (value + tone) — pure presentational |
| Organisms | extend `watchlist-row` / `market-row` with optional `rsiLabel` / `rsiTone` |
| Pages | ViewModel owns batch args, join map, loading flags |

**Do not** put RTK inside atoms/molecules.

---

## 7. Error / empty / loading

| Case | UX |
|------|-----|
| Batch loading, rows already have prices | Row RSI skeleton or muted `…`; do not block whole list |
| Batch HTTP 400/502 | Banner once; rows keep `—` for RSI |
| Per-item `error: unavailable` | That row `—`; others OK |
| Empty favorites | Skip batch call |
| Rate limit 429 | Back off; show “Indicators busy — retry” |

---

## 8. MCP / AI follow-up (not mobile blockers)

| Item | Status |
|------|--------|
| MCP `get_indicators` | Exists (single symbol) |
| MCP `get_indicators_batch` | **Missing** — optional same-MR if agents should bulk-enrich (AGENTS §6.5) |
| OpenAPI | Already documents batch |

Recommend: mobile epic does not block on MCP; track **optional** task or note in epic out-of-scope / follow-up.

---

## 9. Field matrix summary for implementation

### Request defaults (mobile v1)

| Param | Value |
|-------|--------|
| exchange | From row group |
| interval | `1h` |
| rsiPeriod | `14` |
| emaPeriods | `12,26` |
| symbols | unique, ≤50, current enrich set |

### Response used in UI v1

| Field | Use |
|-------|-----|
| `items[].symbol` | Join |
| `items[].rsi` | Badge |
| `items[].error` | Fallback `—` |
| `items[].ema` | Optional / v1.1 |
| `note` | Screen disclaimer once |

---

## 10. Acceptance for this analysis task

- [x] Endpoints + request/response documented  
- [x] Service caps (50), multi-exchange split, partial errors documented  
- [x] Placement: Favorites P0, Markets P1, detail out  
- [x] RTK + Atomic sketch recorded  
- [x] Marked done with implementation

## Status

`done`
