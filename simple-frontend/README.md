# Simple frontend (test harness)

Lightweight static UI for manually exercising the Swyngora market API during development.

> **Not** the production web app. The real product UI will live under `frontend/` (and later may use Atomic Design + RTK Query as described in root `AGENTS.md`).

## Features

- Fetch Binance candlesticks at selectable intervals
- Fetch 24h ticker / base & quote volume
- Fetch circulating / total / max supply (via API → CoinGecko)
- Shows raw JSON for debugging

## Run

1. Start the backend (`backend/`):

   ```bash
   cd backend && go run ./cmd/server
   ```

2. Serve this folder (any static file server). Examples:

   ```bash
   # Python
   cd simple-frontend && python3 -m http.server 5173

   # Node (if npx available)
   npx --yes serve -l 5173 .
   ```

3. Open http://localhost:5173 and click **Fetch all**.

Default API base is `http://localhost:8080` (editable in the UI). CORS is enabled on the backend for local testing.

## Layout

| File | Purpose |
|---|---|
| `index.html` | Page structure |
| `styles.css` | Dark test-console styling |
| `app.js` | Fetch + render logic |
