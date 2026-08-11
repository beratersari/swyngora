# Feature analysis: Coin detail + technical indicators (frontend)

**Status:** Analysis complete (DET-A, DET-B) · **implementation done** (DET-1…DET-4)  
**Surface:** Product web (`frontend/`)  
**Backend:** Existing market APIs — no new endpoints required  
**PM epic:** `project-management/epics/coin-detail-and-indicators.md`

> Process: **PM tasks + analysis first**, implementation only after explicit request.

### Authoritative analysis (full field-level specs)

| Task | Content |
|---|---|
| **[DET-A](../project-management/tasks/frontend/detail/DET-A.md)** | Coin detail: route, orchestration, **intervals / ticker / supply / candles** — every query + response field, mapping, polling, UI binding matrix |
| **[DET-B](../project-management/tasks/frontend/detail/DET-B.md)** | Indicators: **GET /indicators** (all fields) + why **batch** is out; RSI/EMA mapping, bands, overlays, binding matrix |

This file is a short index. **Do not implement from this summary alone** — use DET-A / DET-B.

---

## 1. Problem / goal

Markets list (Epic B) answers “what’s moving.” Users need a **single-pair detail** page: 24h context, supply, OHLCV chart, RSI + EMA on the same interval.

Harness reference only: `simple-frontend/detail.js` (not production Atomic/RTK UI).

---

## 2. Endpoint map (index)

| UI need | Method | Path | Full field spec |
|---|---|---|---|
| Interval dropdown | `GET` | `/api/v1/market/intervals` | DET-A §5 |
| Header + 24h stats | `GET` | `/api/v1/market/ticker/24h` | DET-A §6 |
| Supply stats | `GET` | `/api/v1/market/supply` | DET-A §7 |
| Candle chart | `GET` | `/api/v1/market/candles` | DET-A §8 |
| RSI/EMA series | `GET` | `/api/v1/market/indicators` | DET-B §5 |
| Spot order book | `GET` | `/api/v1/market/orderbook` | [`order-book.md`](order-book.md) |
| Batch (not detail) | `POST` | `/api/v1/market/indicators/batch` | DET-B §6 |

---

## 3. Design decisions (short)

### Page (DET-A)

- Route `/markets/:exchange/:symbol`; query `interval` + `limit` (defaults `1h` / `100`)
- Layout: Header → stats → chart → indicators
- Poll: ticker/supply ~15s; candles ~30s; pause when tab hidden
- Partial errors per section; supply 404 is soft

### Indicators (DET-B)

- Same interval/limit as candles
- RSI: separate 0–100 pane, bands 30/70 (labels educational only)
- EMA: overlays on price chart; periods fixed 12,26; RSI period 14
- Use `GET /indicators` only (not batch) for detail

---

## 4. Implementation order (only if requested)

| ID | Work | Status |
|---|---|---|
| DET-1 | RTK endpoints | done |
| DET-2 | Page shell + route + header/stats | done |
| DET-3 | Candles + toolbar | done |
| DET-4 | Indicator panel + overlays + tests | done |

**Code:** `frontend/src/components/pages/CoinDetailPage/`, `components/organisms/Detail*`, `IndicatorPanel/`, `libs/api/endpoints/marketApi.ts`

---

## 5. Limitations

- Indicators = RSI + EMA only  
- Supply = Binance marketing-list coverage  
- Chart overlays: pump/dump arrows + swing scanner circles (from `/signals` rules)  
- Header links to alerts, compare, and the Signals desk; watchlist star on the pair

DET-1…4 accepted and applied on the product frontend branch.
