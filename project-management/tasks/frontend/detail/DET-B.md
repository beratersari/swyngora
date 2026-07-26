# DET-B: Technical indicators on detail — full analysis (endpoints, fields, logic)

| Field | Value |
|---|---|
| **ID** | DET-B |
| **Epic** | coin-detail-and-indicators |
| **Status** | done (analysis only — **no implementation**) |
| **Area** | frontend / product |
| **Type** | analysis · design · API contract mapping |
| **Depends on** | DET-A (same page, shared `interval` / `limit`) |
| **OpenAPI source** | `backend/api/openapi/openapi.yaml` · operationIds `getIndicators`, `postIndicatorsBatch` |
| **Feature doc** | `docs/features/coin-detail.md` |

---

## 1. Summary

Full analysis of **RSI + EMA** on the coin detail page: layout relative to the candle chart, exact API contracts (fields, defaults, errors), client mapping to chart series, band logic, disclaimers, and what **not** to call (`batch` on detail).

**No implementation** in this task. Implementation = DET-4 (backlog), after DET-1 RTK endpoints.

---

## 2. Problem / goal

| | |
|---|---|
| **Problem** | Candles alone do not surface momentum/trend helpers already available on the API. |
| **Goal** | Show **RSI** and **EMA** for the same symbol/exchange/interval as the price chart. |
| **Honesty** | Informational analysis only — **not financial advice** (product + API `note`). |
| **Constraint** | Use backend math; no client-side RSI/EMA reimplementation as source of truth. |

---

## 3. Placement on the page (with DET-A)

```text
DetailHeader
DetailStats
ChartCard
  ├── Toolbar (interval, limit)     ← shared controls (DET-A)
  └── Candles + optional EMA lines  ← price scale
IndicatorPanel                      ← DET-B
  ├── Snapshot cards (latest RSI / EMAs)
  ├── RSI line chart (0–100, bands 30/70)
  └── Disclaimer / API note
```

| Rule | Spec |
|---|---|
| Same route | No separate `/analysis` route in v1 |
| Shared params | `exchange`, `symbol`, `interval`, `limit` identical to candles request |
| Order | Indicators **below** price chart |
| Independence | Indicators HTTP failure must not unmount candles |

---

## 4. Endpoints overview

| # | Method | Path | Use on detail? |
|---|---|---|---|
| B1 | `GET` | `/api/v1/market/indicators` | **Yes — primary** (series + latest) |
| B2 | `POST` | `/api/v1/market/indicators/batch` | **No on detail** (latest-only, multi-symbol; list/watchlist later) |

Error envelope: same as DET-A (`error.code`, `error.message`).

---

## 5. Endpoint B1 — `GET /api/v1/market/indicators` (detail primary)

### 5.1 Purpose

Server fetches OHLCV for the pair, then computes:

- **RSI** — Wilder’s smoothing (default period 14)  
- **EMA** — default periods 12 and 26; seed = SMA of first period  

Returns **latest snapshot** + full **points** series for charting.

### 5.2 Query parameters

| Name | In | Required | Type | Default | Min | Max | Notes |
|---|---|---|---|---|---|---|---|
| `exchange` | query | no | string | `binance` | | | enum: `binance` \| `coinbase` \| `bybit` |
| `symbol` | query | **yes** | string | — | | | Same venue format as candles (`BTCUSDT` / `BTC-USD`) |
| `interval` | query | no | string | `1h` | | | Must be valid for exchange (see DET-A §5) |
| `limit` | query | no | integer | `100` | 1 | 1000 | **Number of output bars**; backend fetches **extra history** for indicator warm-up |
| `rsiPeriod` | query | no | integer | `14` | 2 | 500 | v1 UI: **always send 14** (or rely on default) |
| `emaPeriods` | query | no | string | `"12,26"` | | | Comma-separated integers; v1 UI: **always `"12,26"`** |

**Detail v1 request (canonical):**

```http
GET /api/v1/market/indicators?exchange={exchange}&symbol={symbol}&interval={interval}&limit={limit}&rsiPeriod=14&emaPeriods=12,26
```

Align `{exchange}`, `{symbol}`, `{interval}`, `{limit}` with the candles request on the same page load.

### 5.3 Response `200` — body fields

| Field | Type | Description |
|---|---|---|
| `exchange` | string | Echo of venue |
| `symbol` | string | Echo of pair |
| `interval` | string | Echo of interval |
| `rsiPeriod` | integer | Effective RSI period used |
| `emaPeriods` | integer[] | Effective EMA periods used (parsed list, e.g. `[12, 26]`) |
| `latest` | object | Most recent computed snapshot (see below) |
| `points` | array | Time series points (see below); length ≈ `limit` (warm-up nulls possible early) |
| `note` | string | Human disclaimer from API (show in UI) |

#### 5.3.1 `latest` object

| Field | Type | Nullable | UI |
|---|---|---|---|
| `latest.rsi` | number \| null | yes | Snapshot card **RSI(rsiPeriod)**; null → show `—` |
| `latest.ema` | object (map string→number) | keys optional | Snapshot cards per period |

**`latest.ema` map semantics:**

| Key | Value type | Meaning | UI |
|---|---|---|---|
| `"12"` | number | EMA period 12 at latest bar | Card + candle overlay series id `ema-12` |
| `"26"` | number | EMA period 26 at latest bar | Card + overlay `ema-26` |
| other keys | number | If API returns more periods later | Sort keys numerically; render card per key |

Keys are **stringified period numbers** (OpenAPI `additionalProperties: number`).

#### 5.3.2 `points[]` item fields

| Field | Type | Nullable | Chart / logic |
|---|---|---|---|
| `openTime` | string (date-time) | should be set | x-axis; parse to UTC seconds |
| `close` | number | | Optional context; **not required** for RSI line |
| `rsi` | number \| null | yes | RSI line **y**; **skip point if null** (warm-up) |
| `ema` | object map string→number | | Per-period price-scale lines; skip missing keys per point |

**Example point (illustrative):**

```json
{
  "openTime": "2026-07-26T12:00:00Z",
  "close": 64500.5,
  "rsi": 58.12,
  "ema": { "12": 64420.1, "26": 64310.0 }
}
```

### 5.4 HTTP errors (UI handling)

| Status | When | UI |
|---|---|---|
| `400` | Bad symbol / interval / periods / limit | Indicator panel Alert; keep candles |
| `404` | Symbol unknown on venue | Same |
| `502` | Upstream candle fetch failed for indicators | Same + retry |
| Network | Offline / CORS / proxy | Generic fetch error via RTK mapper |

Note: OpenAPI lists 400/404/502 for indicators (no 429 on single GET in spec; still handle generically).

### 5.5 Backend behavior relevant to UI (do not re-spec math)

| Behavior | UI implication |
|---|---|
| Extra history for warm-up | Early `points[].rsi` may be `null` — filter out for line chart |
| EMA seeded with SMA | First EMA values appear after period length |
| Informational only | Always show `note` or product disclaimer |
| Interval invalid for venue | 400 — prevented by DET-A interval clamp |

---

## 6. Endpoint B2 — `POST /api/v1/market/indicators/batch` (not used on detail)

Documented so implementers **do not** misuse it on this page.

### 6.1 Purpose

Latest RSI/EMA for **up to 50 symbols**, one exchange/interval — for **Markets columns / watchlist** later.

### 6.2 Request body

| Field | Type | Required | Default | Constraints |
|---|---|---|---|---|
| `symbols` | string[] | **yes** | — | maxItems **50** |
| `exchange` | string | no | `binance` | enum venues |
| `interval` | string | no | `1h` | venue-specific |
| `rsiPeriod` | integer | no | `14` | 2–500 |
| `emaPeriods` | string | no | `"12,26"` | comma-separated |

**Example body:**

```json
{
  "exchange": "binance",
  "interval": "1h",
  "symbols": ["BTCUSDT", "ETHUSDT"],
  "rsiPeriod": 14,
  "emaPeriods": "12,26"
}
```

### 6.3 Response `200`

| Field | Type | Description |
|---|---|---|
| `exchange` | string | Echo |
| `interval` | string | Echo |
| `items` | array | Per-symbol results |
| `note` | string | Disclaimer |

#### `items[]` fields

| Field | Type | Description |
|---|---|---|
| `symbol` | string | Pair |
| `rsi` | number \| null | Latest RSI |
| `ema` | map string→number | Latest EMAs |
| `error` | string | Present on failure; redacted (e.g. `unavailable`) — not full stack traces |

### 6.4 Why detail must not use batch

| Reason | Detail need |
|---|---|
| No `points[]` | RSI **chart** and EMA **overlays** need full series |
| Single symbol | Overhead of batch API unnecessary |
| Per-item soft fail | Detail prefers one clear error for one symbol |

---

## 7. Client mapping logic (pure functions — for later unit tests)

### 7.1 RSI line series

```text
indicatorPointsToRsiLine(points):
  out = []
  for p in points:
    if p.openTime missing → skip
    if p.rsi is null or not finite → skip   // warm-up
    t = floor(Date.parse(p.openTime) / 1000)
    out.push({ time: t, value: p.rsi })
  return out
```

Chart: Lightweight Charts **Line** series, fixed price range **0–100**, horizontal price lines at **30** and **70**.

### 7.2 EMA overlay series (one per period key)

```text
indicatorPointsToEmaLine(points, periodKey):  // e.g. "12"
  out = []
  for p in points:
    v = p.ema?.[periodKey]
    if openTime missing or v missing or not finite → skip
    t = floor(Date.parse(p.openTime) / 1000)
    out.push({ time: t, value: v })
  return out
```

Draw as **Line** series on the **same chart instance as candles** (price scale).  
Color map (v1 recommendation):

| Period key | Color (brand-adjacent) | Legend label |
|---|---|---|
| `12` | mountainMeadow `#4FD4A5` | EMA 12 |
| `26` | frog `#17876D` | EMA 26 |
| other | rotate mint / caribbean / pistachio | EMA {n} |

### 7.3 Latest snapshot formatting

| Value | Formatter | Empty |
|---|---|---|
| RSI | 2 decimal places | `—` if null |
| EMA | 4 decimal places (price-like) | `—` if missing key |

### 7.4 RSI band classification (UI-only; not trading signals)

| Condition on `latest.rsi` | Band label | Text tone |
|---|---|---|
| null / non-finite | `n/a` | secondary |
| `rsi < 30` | `oversold` | success (soft green) |
| `rsi > 70` | `overbought` | error (soft red) |
| else | `neutral` | secondary / mild warning if outside 40–60 optional |

**Copy:** always pair with “informational only — not financial advice.”

### 7.5 Sorted EMA keys

```text
sortedEmaKeys(latest.ema) = Object.keys(ema).sort((a,b) => Number(a) - Number(b))
```

---

## 8. UI binding matrix (DET-B)

| UI element | Source | Field(s) | Logic |
|---|---|---|---|
| Panel title | static | — | “Technical indicators” |
| Period caption | B1 | `rsiPeriod`, `emaPeriods` | `RSI({rsiPeriod}) · EMA({join emaPeriods})` |
| RSI latest value | B1 | `latest.rsi` | `formatIndicator` + band color |
| RSI band label | B1 | `latest.rsi` | `rsiBandLabel` |
| EMA(12) latest | B1 | `latest.ema["12"]` | format 4 dp |
| EMA(26) latest | B1 | `latest.ema["26"]` | format 4 dp |
| RSI chart | B1 | `points[].openTime`, `points[].rsi` | `indicatorPointsToRsiLine` |
| RSI bands 30/70 | static | — | chart price lines |
| EMA overlay on candles | B1 | `points[].ema` | `indicatorPointsToEmaLine` per key; toggle |
| EMA toggle | client state | — | default **on**; does not cancel fetch |
| Disclaimer | B1 | `note` | fallback product string if empty |
| Error Alert | B1 error | RTK error | section-only |

---

## 9. Shared state with DET-A (orchestration)

| Shared input | Owner | Indicators use |
|---|---|---|
| `exchange` | path | B1 query |
| `symbol` | path | B1 query |
| `interval` | URL + A1 clamp | B1 query (**must match candles**) |
| `limit` | URL | B1 query (**same as candles**) |
| `rsiPeriod` | fixed 14 | B1 query |
| `emaPeriods` | fixed `12,26` | B1 query |
| visibility | document | poll 30s / 0 |
| refresh button | toolbar | refetch B1 with A2–A4 |

### 9.1 Request coupling

```text
when interval or limit changes:
  refetch candles (A4) AND indicators (B1) with same params

when only EMA toggle flips:
  do NOT refetch; only show/hide overlay series client-side
```

### 9.2 Polling

| Data | Interval | Pause when hidden |
|---|---|---|
| Indicators (B1) | **30s** (same as candles) | yes |

### 9.3 RTK cache identity (conceptual)

```text
Indicator:{exchange}:{symbol}:{interval}:{limit}:{rsiPeriod}:{emaPeriods}
```

---

## 10. Design decisions (locked for analysis)

| Topic | Decision | Rationale |
|---|---|---|
| Primary API | `GET /indicators` only | Needs `points` |
| Batch API | Not on detail | Latest-only multi-symbol |
| RSI pane | Separate chart, 0–100 | Scale isolation |
| EMA | Overlay on candles + toggle | Price scale |
| Periods v1 | Fixed 14 / 12,26 | Defaults; less UI noise |
| Warm-up nulls | Drop from series; `—` on latest | Correct charts |
| Client math | Forbidden as source of truth | Backend owns formula |
| Compliance | Show API `note` + product footer | AGENTS §6.4 |
| Failure isolation | Indicators error ≠ kill chart | Partial page |

---

## 11. Open questions (product — defaults recommended)

| # | Question | Recommendation |
|---|---|---|
| 1 | Custom RSI/EMA period controls in v1? | **No** — fixed defaults |
| 2 | Oversold = green “opportunity” framing? | Soft green/red **with** disclaimer |
| 3 | Confirm dialog on interval change? | **No** — immediate dual refetch |
| 4 | Volume pane under candles? | **No** in this epic |

---

## 12. Out of scope

- Implementing IndicatorPanel / chart hosts  
- Markets table RSI column via batch (future task)  
- MACD, Bollinger, etc. (no API)  
- Alerts when RSI crosses 30/70  

---

## 13. Acceptance (analysis)

- [x] B1 fully specified: every query param + response field  
- [x] B2 documented as **not for detail**, with full body/fields  
- [x] Mapping logic for RSI line + EMA overlays specified  
- [x] Band labels, formatters, shared orchestration with DET-A  
- [x] UI binding matrix complete  
- [x] Impl task DET-4 remains backlog  

## Status

**done** (analysis). Do not implement without explicit request.
