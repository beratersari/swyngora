# Simple frontend (test harness)

Lightweight static **coin dashboard** for the Swyngora market API during development.

> **Not** the production web app. The real product UI will live under `frontend/` (Atomic Design + RTK Query per root `AGENTS.md`).

## Features

- **Dashboard-first:** markets load on page open (no “list all” click)
- **Live refresh:** auto-poll (default 10s; 5/15/30s or off); pauses when the tab is hidden
- **Price animations:** cells flash green (up) / red (down) when last price, change %, volume, or mcap moves
- **Column sort:** click a header to sort via the API (`sort` / `order`)
- **Watchlist:** star rows (★); chips panel; optional “watchlist only” filter; localStorage + API **union merge** (offline adds are not wiped; local-only items re-POSTed)
- **Indicators:** RSI(14) + EMA(12/26) on the **detail page** only (`/api/v1/market/indicators`); detail loads ignore stale in-flight responses
- **Prices:** tiny non-zero last prices use scientific notation instead of rounding to `0`
- **Column editor:** chips to show/hide; **drag chips or table headers** (or ◀ ▶) to reorder; Symbol stays first (saved in `localStorage`)
- Search + row limit (quote fixed to **USDT**)
- **Double-click** a symbol → `detail.html` (ticker, supply, RSI/EMA, candles)
- Crypto-only list (backend excludes `bStocks` / commodities)

### How fresh is the data?

| Layer | Default |
|---|---|
| Browser poll | every **10s** (configurable in the UI) |
| Backend joined **prices** cache | **5s** (`SPOT_MARKET_CACHE_TTL`) |
| Backend exchangeInfo | ~**10 min** (in-adapter) |
| Backend product tags (crypto filter) | ~**1 hour** (in-adapter) |
| Supply / mcap snapshot | daily (+ startup + failure retries); 48h safety TTL on backend |

Earlier, every price refresh also re-downloaded exchangeInfo + product catalog, so real updates often took **30–40s**. Meta is now cached separately so the 10s poll can show new prices.

## Tests (watchlist logic)

```bash
# from repo root
node --test simple-frontend/watchlist-logic.test.js
```

Covers: full membership (6 coins), regression of “filter top-N page”, sort does not change count, exact symbol pick, placeholders for missing markets, watchlist merge, fmtNum for tiny prices.

## Run

1. Start the backend (`backend/`):

   ```bash
   cd backend && go run ./cmd/server
   ```

2. Serve this folder:

   ```bash
   cd simple-frontend && python3 -m http.server 5173
   ```

3. Open http://localhost:5173 — markets load automatically.

Default API base is `http://localhost:8080` (editable in the header).

## Layout

| File | Purpose |
|---|---|
| `index.html` | Dashboard shell |
| `styles.css` | Dark dashboard styling |
| `app.js` | Dashboard: markets, sort, columns, watchlist, live poll |
| `detail.js` | Coin detail page (ticker, supply, indicators, candles) |
| `watchlist-logic.js` | Pure helpers (merge, fmtNum, watchlist assembly) + unit tests |

- **Detail page:** double-click a symbol → `detail.html?symbol=…&exchange=…`
