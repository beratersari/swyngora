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
| Holder snapshot | `GET` | `/api/v1/market/holders` | [`holders.md`](holders.md) |
| Candle chart | `GET` | `/api/v1/market/candles` | DET-A §8 |
| RSI/EMA series | `GET` | `/api/v1/market/indicators` | DET-B §5 |
| Spot order book | `GET` | `/api/v1/market/orderbook` | [`order-book.md`](order-book.md) |
| Market depth graph | `GET` | `/api/v1/market/orderbook` (same live book) | [`order-book.md`](order-book.md) |
| Order heatmap | `GET` | `/api/v1/market/orderbook/heatmap` | [`order-book.md`](order-book.md) |
| Batch (not detail) | `POST` | `/api/v1/market/indicators/batch` | DET-B §6 |

---

## 3. Design decisions (short)

### Page (DET-A)

- Route `/markets/:exchange/:symbol`; query `interval` + `limit` (defaults `1h` / `100`)
- Layout: Header → stats → tabs (Overview chart · Order book · Holders · Indicators · Trade)
- Poll: ticker/supply ~15s; candles ~30s; pause when tab hidden
- Chart starts the kline fetch immediately (does not wait for the intervals list). A 100-bar first request paints the viewport, then the 300-bar live window replaces it. EMA overlays come from loaded candles so they do not block first paint. The overview chart stays mounted when switching detail tabs.
- Partial errors per section; supply 404 is soft

### Indicators (DET-B)

- Same interval/limit as candles
- RSI: separate 0–100 pane, bands 30/70 (labels educational only)
- EMA: overlays on the price chart from **loaded candles** (same SMA-seed EMA as the backend) so pan-left history keeps the line; periods 12,26
- RSI snapshot on the Indicators tab still uses `GET /indicators` (not batch)

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
- Holders = CoinMarketCap public snapshot when published; 404 otherwise  
- Chart overlays: pump/dump arrows + swing scanner circles (from `/signals` rules). Live pump markers use the current pair only (`rtkCurrent` + exchange/symbol match) so a previous coin’s events cannot snap onto the new candles.  
- Header links to alerts, compare, and the Signals desk; watchlist star on the pair

DET-1…4 accepted and applied on the product frontend branch.
