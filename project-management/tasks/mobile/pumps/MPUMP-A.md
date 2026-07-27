# MPUMP-A: Pump API field matrix (analysis)

| Field | Value |
|---|---|
| **ID** | MPUMP-A |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/pumps/MPUMP-A.md` |

## Purpose

Map backend pump HTTP contracts to mobile UI needs. **No backend work required** — APIs exist.

Sources:

- OpenAPI: `backend/api/openapi/openapi.yaml` (`getPumpEvents`, `scanPumpEvents`)
- Handler DTOs: `backend/internal/transport/http/handler/pumps.go`
- Domain: `backend/internal/domain/pump.go`
- Service: `backend/internal/service/market/pumps.go`
- MCP: `detect_pump_events`, `scan_pump_events`

---

## 1. Endpoints

| Method | Path | operationId | Mobile use |
|--------|------|-------------|------------|
| `GET` | `/api/v1/market/pumps` | `getPumpEvents` | Coin detail “Pumps” section / event timeline |
| `GET` | `/api/v1/market/pumps/scan` | `scanPumpEvents` | **Pumps** tab — market-wide radar |

Both are **read-only**, mechanical threshold detection over OHLCV. Responses include a **note**: informational only — not financial advice.

---

## 2. Single-symbol — `GET /api/v1/market/pumps`

### Query parameters

| Param | Required | Default (service) | Type | UI |
|-------|----------|-------------------|------|-----|
| `symbol` | **yes** | — | string | From route (`BTCUSDT`) |
| `exchange` | no | `binance` | enum binance\|coinbase\|bybit | From markets/detail |
| `interval` | no | `1h` | string (venue-valid) | Interval chips; validate via `/intervals` |
| `lookbackHours` | no | (if unset, use `limit`) | number | Optional range control |
| `limit` | no | `100` bars (max 1000) | int | When lookbackHours not set |
| `minReturnPct` | no | **5** | number > 0 | Threshold slider/stepper |
| `windowBars` | no | **1** (max 100) | int | Advanced; default hide |
| `mode` | no | `close_return` | enum | Advanced mode picker |
| `direction` | no | `up` | enum up\|down\|both | Direction chips |
| `minVolumeRatio` | no | 0 = off | number ≥ 0 | Optional volume filter |
| `maxEvents` | no | **20** | int | Cap list length |
| `startTime` / `endTime` | no | — | RFC3339 / unix | Out of scope v1 |

### Response fields (handler)

| Field | Type | UI mapping |
|-------|------|------------|
| `exchange` | string | Header badge |
| `symbol` | string | Header |
| `interval` | string | Subtitle |
| `lookbackHours` | number | Meta |
| `barsAnalyzed` | int | Meta (“N bars”) |
| `minReturnPct` | number | Show active threshold |
| `windowBars` | int | Advanced meta |
| `mode` | string | Advanced meta |
| `direction` | string | Filter state |
| `eventCount` | int | Count chip |
| `events[]` | array | Event list |
| `note` | string | Footer disclaimer |

### `events[]` item

| Field | Type | UI |
|-------|------|-----|
| `index` | int | Internal / debug |
| `openTime` | RFC3339Nano | Event time label |
| `closeTime` | RFC3339Nano | Optional |
| `startPrice` | float | From price |
| `endPrice` | float | To price |
| `returnPct` | float signed | **Primary** (+ pump / − dump), color tone |
| `high` / `low` | float | Optional detail |
| `volume` | float | Secondary |
| `volumeRatio` | float | “× median vol” if > 0 |
| `mode` | string | Chip |
| `windowBars` | int | Meta |

---

## 3. Scan — `GET /api/v1/market/pumps/scan`

### Query parameters

| Param | Default (service) | Notes | UI |
|-------|-------------------|-------|-----|
| `exchange` | binance | Venue | Exchange chips |
| `quote` | USDT (USD on Coinbase) | Spot filter | Quote chip |
| `interval` | **15m** | Stricter than single-symbol 1h | Interval picker |
| `lookbackHours` | **24** | Scan window | Lookback chips (6h/24h/48h) |
| `minReturnPct` | **8** | Higher than single 5 | Threshold control |
| `windowBars` | 1 | Advanced | Optional |
| `mode` | close_return | Advanced | Optional |
| `direction` | up | up\|down\|both | Direction chips |
| `minVolumeRatio` | 0 | Optional | Advanced |
| `symbolLimit` | **15** (max **40**) | Top quote-volume symbols scanned | “Depth” control |
| `maxTotalEvents` | 30 (handler accepts; service default 30) | Cap hits | Internal |

**Concurrency:** service scans with semaphore of **5** concurrent candle fetches. Expect multi-second latency on cold scan — mobile must show loading + allow cancel/unmount.

### Response fields

| Field | Type | UI |
|-------|------|-----|
| `exchange` | string | May be empty string if query omitted — prefer client-known exchange |
| `interval` | string | Subtitle |
| `lookbackHours` | number | Meta |
| `minReturnPct` | number | Meta |
| `hitCount` | int | Count |
| `hits[]` | array | Ranked list |
| `note` | string | Disclaimer |

### `hits[]` item

| Field | Type | UI |
|-------|------|-----|
| `symbol` | string | Primary label |
| `exchange` | string | Badge |
| `interval` | string | Meta |
| `bestReturnPct` | float | **Sort key display** (abs rank server-side) |
| `events[]` | same as above | Expandable / first event summary |

Hits sorted by **\|bestReturnPct\|** descending.

---

## 4. Detect modes (domain)

| Mode | Meaning | Default UX |
|------|---------|------------|
| `close_return` | close[i−window] → close[i] % | **Default** — clearest |
| `candle_body` | open → close of bar | Advanced |
| `high_from_low` | low → high of bar | Advanced (wick spike) |

Direction: `up` (pumps), `down` (dumps), `both`.

---

## 5. OpenAPI gaps / client notes

1. OpenAPI schemas for `events` / `hits` items are loosely `object` — **use handler DTOs** as the practical contract until OpenAPI is tightened (optional follow-up backend task, not blocking mobile).
2. Scan response `exchange` echoes query param (can be empty) — client should keep selected exchange from UI state.
3. Coinbase default quote for scan is **USD** when quote empty; mobile should pass `quote=USD` on Coinbase explicitly.
4. Intervals must be valid per exchange — call `GET /intervals?exchange=` (already on mobile).
5. Rate limits + multi-symbol scan = **no aggressive polling**; pull-to-refresh + manual only for v1.

---

## 6. Mobile screen mapping

| Screen | Endpoint | Defaults |
|--------|----------|----------|
| **Pumps tab** (scan list) | `/pumps/scan` | exchange=binance, quote=USDT, interval=15m, lookback=24h, minReturnPct=8, direction=up, symbolLimit=15 |
| **Coin detail → Pumps section** | `/pumps` | symbol+exchange from route, interval=1h (or detail interval), minReturnPct=5, direction=both, maxEvents=10 |
| Row press | — | Navigate to CoinDetail `{ exchange, symbol }` |

---

## 7. Error / empty UX

| Case | Behavior |
|------|----------|
| 400 invalid interval/param | Inline error + fix controls |
| 502 upstream | Retry button |
| `hitCount=0` / `eventCount=0` | Empty: “No pumps matched thresholds” |
| Slow scan | Skeleton + cancel on leave (RTK abort) |

---

## Status

Analysis complete — ready for MPUMP implementation tasks.
