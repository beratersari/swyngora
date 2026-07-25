# Simple frontend (test harness)

Lightweight static **coin dashboard** for the Swyngora market API during development.

> **Not** the production web app. The real product UI will live under `frontend/` (Atomic Design + RTK Query per root `AGENTS.md`).

## Features

- **Dashboard-first:** markets load on page open (no “list all” click)
- **Live refresh:** auto-poll (default 10s; 5/15/30s or off); pauses when the tab is hidden
- **Price animations:** cells flash green (up) / red (down) when last price, change %, volume, or mcap moves
- **Column sort:** click a header to sort via the API (`sort` / `order`)
- **Column editor:** always-visible chips to show/hide and drag-reorder columns (saved in `localStorage`)
- Search + quote filter + row limit
- Row / symbol click → detail (24h ticker, supply, candles)
- Crypto-only list (backend excludes `bStocks` / commodities)

### How fresh is the data?

| Layer | Default |
|---|---|
| Browser poll | every **10s** (configurable in the UI) |
| Backend joined **prices** cache | **5s** (`SPOT_MARKET_CACHE_TTL`) |
| Backend exchangeInfo | ~**10 min** (in-adapter) |
| Backend product tags (crypto filter) | ~**1 hour** (in-adapter) |
| Supply / mcap snapshot | daily (+ startup), not tick-by-tick |

Earlier, every price refresh also re-downloaded exchangeInfo + product catalog, so real updates often took **30–40s**. Meta is now cached separately so the 10s poll can show new prices.

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
| `app.js` | Auto-load, sort, columns, detail panel |
