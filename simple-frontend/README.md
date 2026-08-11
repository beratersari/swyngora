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
- **AI chat:** `ai.html` → `POST /api/v1/ai/chat` (multi-turn `sessionId`; needs Python AI on `:8090`); **Markdown** replies rendered via `markdown.js`
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
node --test simple-frontend/markdown.test.js
```

Covers: full membership (6 coins), regression of “filter top-N page”, sort does not change count, exact symbol pick, placeholders for missing markets, watchlist merge, fmtNum for tiny prices; Markdown escape/lists/tables/code.

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
4. **AI chat:** http://localhost:5173/ai.html (also linked as **AI chat** in the header).

Default API base is `http://localhost:8080` (editable in the header).

### AI chat prerequisites

```bash
# Backend already running on :8080
# Python AI (Ollama or Grok):
cd ai && uv sync && source .venv/bin/activate
export SWYNGORA_API_URL=http://127.0.0.1:8080
python -m swyngora_ai.serve --host 127.0.0.1 --port 8090
```

Without the AI service, the page still works; replies show an upstream/unavailable error.

## Layout

| File | Purpose |
|---|---|
| `index.html` | Dashboard shell |
| `styles.css` | Dark dashboard styling |
| `app.js` | Dashboard: markets, sort, columns, watchlist, live poll |
| `detail.html` / `detail.js` | Coin detail (ticker, supply, indicators, candles) |
| `ai.html` / `ai.js` | AI multi-agent chat harness |
| `markdown.js` | Safe Markdown → HTML for assistant bubbles |
| `watchlist-logic.js` | Pure helpers (merge, fmtNum, watchlist assembly) + unit tests |

- **Detail page:** double-click a symbol → `detail.html?symbol=…&exchange=…`
- **AI page:** header **AI chat** → `ai.html` (optional `?q=` prefill)
