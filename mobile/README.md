# Swyngora Mobile

React Native product client for Swyngora — **web-first scaffold** (Chrome) with **no Expo**.

Source follows Atomic Design + **modules that own pages** + **View + ViewModel**. Brand colors match `frontend/src/styles/tokens/colors.ts`.

## Run in Chrome (primary)

```bash
cd mobile
npm install
npm run web
```

Open **http://localhost:5180** (or the WSL IP printed by Vite, e.g. `http://172.x.x.x:5180` from Windows Chrome).

| Script | Purpose |
|--------|---------|
| `npm run web` / `npm run dev` | Vite + **react-native-web** → Chrome |
| `npm run build:web` | Production web build |
| `npm test` | Vitest |
| `npm run lint` | ESLint (incl. import boundaries) |
| `npm run typecheck` | `tsc` |
| `npm run codegen:api` | OpenAPI → `src/libs/api/generated/` |

Backend (optional for Home health):

```bash
cd ../backend && go run ./cmd/server
```

Vite proxies `/api` and `/health` to `http://127.0.0.1:8080` (override with `VITE_DEV_PROXY_TARGET`).

## Why Chrome, not a simulator?

This init targets **browser development** via **react-native-web** so you can use the app without Android/iOS simulators. UI is written with React Native primitives (`View`, `Text`, `StyleSheet`) so the same screens can later target native.

`npm run android` / `npm run ios` are **not** wired yet (native `android/` / `ios/` folders are not generated). Use `npm run web`.

## Architecture

```text
src/
  app/                 # providers + React Navigation
  components/          # Atomic ONLY (no pages)
  modules/
    app/pages/         # HomePage (View + ViewModel)
    markets/pages/     # Markets + Coin detail
    watchlist/pages/   # Watchlist tab
  libs/api/            # RTK Query + OpenAPI
  styles/tokens/       # same brand hex as frontend
```

See `AGENTS.md` and `docs/design/mobile-project-initialization.md`.



## Coin detail

Tap a market row to open **Coin detail**:

- Header + 24h change
- Stats (OHLC, volume, supply)
- Interval chips + EMA toggle
- OHLCV chart (Lightweight Charts on web) + RSI pane

```text
modules/markets/pages/CoinDetailPage/
components/organisms/{CoinDetailHeader,CoinDetailStats,IntervalToolbar,CandleChart,IndicatorRsiPane}
```

## Multi-exchange markets dashboard

The **Markets** tab loads live spot markets from the backend:

- Exchange chips: Binance / Coinbase / Bybit
- Search (300ms debounce), quote, sort, infinite scroll (page size 30)
- **Filter tags** on a dedicated screen (Filter by tags)
- Skeleton footer while loading the next page on scroll
- Module **React context** for shared filter state (`MarketsProvider`)
- ~10s polling on the first page while focused
- **★ star** on each row to add/remove from the Watchlist tab

## Watchlist

The **Watchlist** tab lists pairs you starred:

- Star/unstar from Markets rows or Coin detail header
- Device-local `clientId` (`mobile-<uuid>`) + optimistic sync to `GET/POST/DELETE /api/v1/watchlist*`
- 24h ticker quotes while the tab is focused
- Local cache survives backend restarts (re-POSTs missing items on hydrate)

```text
modules/watchlist/
libs/api/endpoints/watchlistApi.ts
components/molecules/StarButton
```

Feature: `docs/features/mobile-watchlist.md`
- Pull-to-refresh resets the list

```bash
# terminal 1
cd backend && go run ./cmd/server

# terminal 2
cd mobile && npm run web
# open Markets tab in Chrome
```

Architecture: `src/modules/markets/` (View + ViewModel + module components).  
Data: `src/libs/api/endpoints/marketApi.ts`.  
Design: `docs/design/mobile-markets-dashboard.md`.

## Color tokens

Copied from frontend:

| Token | Hex |
|-------|-----|
| navy | `#111844` |
| indigo | `#4B5694` |
| steel | `#7288AE` |
| cream | `#EAE0CF` |

## Env

| Variable | Meaning |
|----------|---------|
| `VITE_API_BASE_URL` | Optional absolute API origin. **Empty (default)** = same origin + Vite proxy (best for Chrome on WSL). |
| `VITE_DEV_PROXY_TARGET` | Vite-only proxy target (default `http://127.0.0.1:8080`). |

## Toolchain pins

| Tool | Notes |
|------|--------|
| Node | 20+ |
| Package manager | npm |
| Bundler (web) | Vite 6 |
| UI runtime | react-native-web (aliases `react-native`) |
| Navigation | React Navigation 7 |
| State | Redux Toolkit + RTK Query |

**Last updated:** 2026-07-26
