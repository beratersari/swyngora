# DET-A: Coin detail page — full analysis (endpoints, fields, logic)

| Field | Value |
|---|---|
| **ID** | DET-A |
| **Epic** | coin-detail-and-indicators (`project-management/epics/coin-detail-and-indicators.md`) |
| **Status** | done (analysis only — **no implementation**) |
| **Area** | frontend / product |
| **Type** | analysis · design · API contract mapping |
| **OpenAPI source** | `backend/api/openapi/openapi.yaml` |
| **Companion analysis** | DET-B (indicators on this page) |
| **Feature doc** | `docs/features/coin-detail.md` |

---

## 1. Summary

Full product analysis for the **coin detail** page: navigation, URL state, section layout, every HTTP endpoint with **query/response fields**, client mapping logic, error/loading/polling rules, and formatters.  

**This task does not implement UI or RTK.** Implementation is DET-1…DET-3 (backlog).

---

## 2. Problem / goal

| | |
|---|---|
| **Problem** | Markets list (Epic B) shows many pairs but not deep context for one symbol. |
| **Goal** | Single-pair page: identity + last price, 24h stats, supply context, OHLCV chart. |
| **User path** | Markets table → open pair → inspect stats/chart → (indicators: DET-B). |
| **Constraint** | No new backend endpoints; OpenAPI contract as-is. |

---

## 3. Information architecture

### 3.1 Route

| Piece | Spec |
|---|---|
| **Path** | `/markets/:exchange/:symbol` |
| **`:exchange`** | One of `binance` \| `coinbase` \| `bybit` (case-insensitive parse → lowercase). Invalid → treat as `binance` or show 404-style message. |
| **`:symbol`** | Trading pair as returned by that venue (e.g. `BTCUSDT`, `BTC-USD`). Decode URI; normalize display to uppercase. |
| **Encoding** | When navigating: `encodeURIComponent(exchange)`, `encodeURIComponent(symbol)`. |
| **Example** | `/markets/binance/BTCUSDT`, `/markets/coinbase/BTC-USD` |

### 3.2 Query string (client state)

| Query key | Type | Default | Allowed | Persist when default? |
|---|---|---|---|---|
| `interval` | string | `1h` | Must be in venue list from `GET /intervals` | **No** (omit if `1h`) |
| `limit` | integer | `100` | UI options: `50`, `100`, `200`, `500` (API allows 1–1000) | **No** (omit if `100`) |

Not in URL v1: indicator periods (fixed in DET-B), poll interval, chart zoom.

### 3.3 Entry / exit

| Action | Behavior |
|---|---|
| **Entry from Markets** | Row click (or keyboard later): navigate to `/markets/{exchange}/{symbol}` preserving **detail** defaults for interval/limit (list query params are **not** required on detail v1). |
| **Back** | Link “← Markets” → `/markets` (optionally restore list search params in a later polish). |
| **Header brand** | Link to `/markets`. |

### 3.4 Page sections (top → bottom)

```text
1. DetailHeader     — identity + last + 24h %
2. DetailStats      — ticker grid + supply grid
3. ChartCard
   ├── Toolbar      — interval, limit, refresh, fetch status
   └── Candle chart — Lightweight Charts OHLCV
4. IndicatorPanel   — DET-B (same page; same interval/limit)
```

---

## 4. Endpoints used by coin detail (DET-A scope)

DET-A owns **intervals, ticker, supply, candles**. DET-B owns **indicators** (listed here only for orchestration).

| # | Method | Path | operationId | Section |
|---|---|---|---|---|
| A1 | `GET` | `/api/v1/market/intervals` | `listIntervals` | Toolbar interval options |
| A2 | `GET` | `/api/v1/market/ticker/24h` | `getTicker24h` | Header + stats (24h) |
| A3 | `GET` | `/api/v1/market/supply` | `getSupply` | Header name + supply stats |
| A4 | `GET` | `/api/v1/market/candles` | `getCandles` | Price chart |
| B1 | `GET` | `/api/v1/market/indicators` | `getIndicators` | DET-B only |

**Error envelope (all failures unless noted):**

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

| HTTP | Typical meaning for UI |
|---|---|
| `400` | Bad symbol/interval/limit/exchange |
| `404` | Unknown symbol (ticker/candles) or no supply snapshot |
| `429` | Rate limited — show retry message |
| `502` | Upstream exchange / supply path failure |

---

## 5. Endpoint A1 — `GET /api/v1/market/intervals`

### 5.1 Purpose

Populate the **interval** dropdown for the selected exchange. Coinbase/Bybit reject many Binance-only intervals with `400` if called on candles without clamping.

### 5.2 Query parameters

| Name | In | Required | Type | Default | Enum / notes |
|---|---|---|---|---|---|
| `exchange` | query | no | string | `binance` | `binance` \| `coinbase` \| `bybit` |

**Example:** `GET /api/v1/market/intervals?exchange=coinbase`

### 5.3 Response `200`

| Field | Type | Required (practical) | Description |
|---|---|---|---|
| `exchange` | string | yes | Echo of resolved venue |
| `intervals` | string[] | yes | Ordered list of supported interval ids |

**Venue interval sets (backend domain):**

| Exchange | `intervals` |
|---|---|
| **binance** | `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M` |
| **coinbase** | `1m`, `5m`, `15m`, `1h`, `6h`, `1d` |
| **bybit** | `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `12h`, `1d`, `1w`, `1M` |

### 5.4 Client logic

1. On mount / when `exchange` path param changes → fetch intervals.  
2. Let `requested = URL interval || "1h"`.  
3. **Resolve:**  
   - if `requested ∈ intervals` → use it  
   - else if `"1h" ∈ intervals` → use `1h` and **replace** URL  
   - else use `intervals[0]` and **replace** URL  
4. Do **not** call candles/indicators until interval is resolved (or call once and accept 400 only if race — prefer wait).  
5. Cache via RTK (`providesTags: Interval` by exchange); revalidate on exchange change.

---

## 6. Endpoint A2 — `GET /api/v1/market/ticker/24h`

### 6.1 Purpose

Header last price / 24h change and stats grid OHLC + volumes + trades.

### 6.2 Query parameters

| Name | In | Required | Type | Default | Notes |
|---|---|---|---|---|---|
| `exchange` | query | no | string | `binance` | enum: binance, coinbase, bybit |
| `symbol` | query | **yes** | string | — | Venue pair format |

**Example:** `GET /api/v1/market/ticker/24h?exchange=binance&symbol=BTCUSDT`

### 6.3 Response `200` — schema `Ticker24h`

| Field | Type | UI use | Format notes |
|---|---|---|---|
| `exchange` | string | debug / verify | |
| `symbol` | string | verify path match | |
| `lastPrice` | string (decimal) | **Header primary** | Use `formatPrice` (scientific for tiny) |
| `priceChange` | string | optional secondary | Absolute 24h change |
| `priceChangePercent` | string | **Header %** + color | `formatChangePercent`; green if &gt;0, red if &lt;0 |
| `openPrice` | string | Stats: Open | |
| `highPrice` | string | Stats: High 24h | Coinbase may fill high/low from secondary stats API server-side |
| `lowPrice` | string | Stats: Low 24h | |
| `volume` | string | Stats: Base vol | Base asset volume 24h |
| `quoteVolume` | string | Stats: Quote vol | Quote asset volume 24h (compact USD-style) |
| `openTime` | string (date-time) | optional tooltip | RFC3339 |
| `closeTime` | string (date-time) | optional tooltip | RFC3339 |
| `tradeCount` | integer (int64) | Stats: Trades | Show `—` if `0` and exchange ≠ binance |

### 6.4 Client logic

| Step | Rule |
|---|---|
| Request | `{ exchange, symbol }` from path |
| Poll | every **15s** while document visible; **0** when hidden; `refetchOnFocus: true` |
| Loading | skeleton on header/stats until first success |
| Error | Section warning; do **not** blank chart if candles still OK |
| Stale | Keep last good ticker while refetching |

---

## 7. Endpoint A3 — `GET /api/v1/market/supply`

### 7.1 Purpose

Asset name (header), circulating/total/max supply, optional USD snapshot price. Source: Binance marketing list cache (not per-request Spot API).

### 7.2 Query parameters

| Name | In | Required | Type | Notes |
|---|---|---|---|---|
| `asset` | query | conditional | string | Base ticker preferred, e.g. `BTC` |
| `symbol` | query | conditional | string | Pair allowed, e.g. `BTCUSDT`; stable quote suffix stripped server-side |

**Rule:** at least one of `asset` or `symbol` required.

**Detail v1 recommendation:** pass `symbol=<path symbol>` so Coinbase `BTC-USD` still resolves when backend strips quote (verify edge cases; if 404, show soft empty supply).

**Example:** `GET /api/v1/market/supply?symbol=BTCUSDT`

### 7.3 Response `200` — schema `Supply`

| Field | Type | Nullable | UI use |
|---|---|---|---|
| `asset` | string | | Base id (e.g. BTC) |
| `name` | string | | **Header subtitle** if present |
| `providerId` | string | | Internal; hide or tooltip only |
| `circulatingSupply` | number \| null | yes | Stats: Circ. supply |
| `totalSupply` | number \| null | yes | Stats: Total supply |
| `maxSupply` | number \| null | yes | Stats: Max; **null** = no hard cap → display `∞ / n/a` |
| `currentPriceUsd` | number \| null | yes | Stats: USD from supply snap (may differ slightly from live last) |
| `asOf` | string (date-time) | | Caption “as of …” for staleness |
| `source` | string | | Expect `binance`; caption |
| `note` | string | | Optional API note |

### 7.4 Client logic

| Step | Rule |
|---|---|
| Parallel | Fetch **with** ticker on load (do not block chart) |
| `404` | Soft info alert: “Supply snapshot not available for this asset” — not a full-page error |
| Poll | same **15s** / visibility rules as ticker (snapshot changes slowly; light poll OK) |
| Format supply | Compact units (K/M/B/T) **without** `$` prefix |
| Format USD | `formatPrice` |

---

## 8. Endpoint A4 — `GET /api/v1/market/candles`

### 8.1 Purpose

OHLCV series for Lightweight Charts candlesticks.

### 8.2 Query parameters

| Name | In | Required | Type | Default | Min | Max | Notes |
|---|---|---|---|---|---|---|---|
| `exchange` | query | no | string | `binance` | | | enum venues |
| `symbol` | query | **yes** | string | | | | venue pair |
| `interval` | query | no | string | `1h` | | | must be valid **for exchange** |
| `limit` | query | no | integer | `100` | 1 | 1000 | bar count |
| `startTime` | query | no | string | | | | RFC3339 or unix **ms** — **v1 UI does not send** |
| `endTime` | query | no | string | | | | RFC3339 or unix **ms** — **v1 UI does not send** |

**Example:**  
`GET /api/v1/market/candles?exchange=binance&symbol=BTCUSDT&interval=1h&limit=100`

### 8.3 Response `200` — schema `CandlesResponse`

| Field | Type | Description |
|---|---|---|
| `symbol` | string | Echo |
| `interval` | string | Echo |
| `exchange` | string | Echo |
| `candles` | `Candle[]` | Time-ordered series (oldest → newest expected; client must not assume reverse) |

#### `Candle` item fields

| Field | Type | Chart use | Notes |
|---|---|---|---|
| `openTime` | string (date-time) | **x-axis time** | Parse → UTC seconds for Lightweight Charts |
| `open` | string (decimal) | open | `Number(...)` |
| `high` | string (decimal) | high | |
| `low` | string (decimal) | low | |
| `close` | string (decimal) | close | |
| `volume` | string (decimal) | optional volume pane later | v1: unused on chart |
| `closeTime` | string (date-time) | unused v1 | |
| `quoteVolume` | string | unused v1 | |
| `tradeCount` | integer | unused v1 | |

### 8.4 Mapping logic (client pure function)

```text
apiCandlesToChart(candles):
  for each candle:
    if any of openTime/open/high/low/close missing or non-finite → skip
    time  = floor(Date.parse(openTime) / 1000)   // UTCTimestamp seconds
    open/high/low/close = Number(string)
  return ChartCandle[]
```

Lightweight Charts candlestick bar: `{ time, open, high, low, close }`.

### 8.5 Client logic

| Step | Rule |
|---|---|
| Depends on | Resolved `interval` + path `exchange`/`symbol` + URL `limit` |
| Poll | **30s** while visible (backend candle TTL ~30s) |
| Empty series | Empty state “No candle data” |
| Error | Chart card Alert + retry; keep header if ticker OK |
| Interval change | Update URL → RTK new cache key → refetch; `fitContent` after setData |
| Limit change | Same |

---

## 9. Orchestration logic (page load)

### 9.1 Inputs

```text
path:  exchange, symbol
query: interval?, limit?
```

### 9.2 Sequence

```text
1. Parse exchange (enum) + symbol (decode, non-empty).
2. If symbol empty → hard error state.
3. Fetch intervals(exchange)  [A1]
4. Resolve interval (clamp) → maybe rewrite URL.
5. In parallel:
   - ticker(exchange, symbol)           [A2]  poll 15s
   - supply(symbol)                     [A3]  poll 15s; soft fail
   - candles(exchange, symbol, iv, lim) [A4]  poll 30s
   - indicators(...)                    [B1]  DET-B; poll 30s
6. Map candles → chart; map ticker/supply → header/stats.
7. On interval/limit change: cancel obsolete UI paint (RTK cache key change is enough; avoid stale paint if using manual seq).
8. On document hidden: set all pollingInterval = 0; on visible: restore + optional refetchOnFocus.
```

### 9.3 RTK cache keys (logical)

| Query | Cache identity (conceptual) |
|---|---|
| intervals | `Interval:{exchange}` |
| ticker | `Ticker:{exchange}:{symbol}` |
| supply | `Supply:{symbol|asset}` |
| candles | `Candle:{exchange}:{symbol}:{interval}:{limit}` |

### 9.4 Manual refresh

Toolbar **Refresh** refetches A2+A3+A4 (+ B1). Does not clear interval.

---

## 10. UI field binding matrix (DET-A)

| UI element | Source endpoint | Source field(s) | Formatter / logic |
|---|---|---|---|
| Back link | — | — | route `/markets` |
| Symbol title | path | `symbol` | uppercase display |
| Exchange tag | path | `exchange` | lowercase tag |
| Asset name | A3 | `name` | hide if empty |
| Last price | A2 | `lastPrice` | `formatPrice` |
| 24h % | A2 | `priceChangePercent` | `formatChangePercent` + tone |
| Open | A2 | `openPrice` | `formatPrice` |
| High 24h | A2 | `highPrice` | `formatPrice` |
| Low 24h | A2 | `lowPrice` | `formatPrice` |
| Base vol | A2 | `volume` | `formatCompactUsd` or compact number |
| Quote vol | A2 | `quoteVolume` | `formatCompactUsd` |
| Trades | A2 | `tradeCount` | `formatTradeCount(exchange)` |
| Circ. supply | A3 | `circulatingSupply` | compact count |
| Total supply | A3 | `totalSupply` | compact count |
| Max supply | A3 | `maxSupply` | null → `∞ / n/a` |
| USD snap | A3 | `currentPriceUsd` | `formatPrice` |
| Supply asOf | A3 | `asOf` | locale date caption |
| Interval select | A1 | `intervals[]` | options; value = resolved interval |
| Bars select | URL | `limit` | 50/100/200/500 |
| Candles chart | A4 | `candles[]` | `apiCandlesToChart` |

---

## 11. Design decisions (summary)

| Topic | Decision |
|---|---|
| Route | `/markets/:exchange/:symbol` |
| Chart lib | Lightweight Charts (locked) |
| Range queries | Not in v1 (`startTime`/`endTime` unused) |
| Partial failure | Per-section errors |
| Polling | 15s ticker/supply; 30s candles; pause when hidden |
| Atomic | Page owns RTK; feature components presentational |
| Watchlist | Out of scope (WL-1) |

---

## 12. Out of scope (DET-A)

- RSI/EMA UI (DET-B)  
- Watchlist  
- `POST /indicators/batch`  
- Paper trading / alerts  
- Implementation of React components  

---

## 13. Acceptance (analysis)

- [x] Route, query state, section layout specified  
- [x] All DET-A endpoints with **every** query + response field documented  
- [x] Client mapping, polling, error, clamp logic specified  
- [x] UI binding matrix complete  
- [x] Impl split DET-1…DET-3 backlog  

## Status

**done** (analysis). Do not implement without explicit request.
