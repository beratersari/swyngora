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
| `npm test` | Vitest (includes `src/libs/api/baseApi.auth.e2e.test.ts` against a local HTTP server) |
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
  components/          # Atomic ONLY (no pages); kebab-case folders
  modules/
    app/pages/         # home-page (View + ViewModel)
    markets/pages/     # markets-page, coin-detail-page, …
    watchlist/pages/   # watchlist-page (Favorites tab)
    ai/pages/          # ai-chat-page (Ask tab)
  libs/api/            # RTK Query + OpenAPI
  libs/i18n/           # i18next catalogs (en/tr), useLocale, LanguageSwitcher
  styles/tokens/       # same brand hex as frontend
```

## Icons

**Lucide** (`lucide-react-native` + `react-native-svg`) for tab bar, stars, filters, back, language.

```tsx
import { Star } from 'lucide-react-native';
import { Icon } from '@/components/atoms/icon';

<Icon icon={Star} size="md" color="#F5C542" fill="#F5C542" />
```

## Localization

- **i18next** + **react-i18next**; languages **en** + **tr** (extensible)
- Catalogs: `src/libs/i18n/locales/<lng>/{common,home,markets,watchlist,pumps,detail,ai}.json`
- Switch language on **Home** via Language chips (persisted in localStorage)
- Add a locale: new folder + register in `resources.ts` + `SUPPORTED_LOCALES` (see `libs/i18n/README.md`)

See `AGENTS.md` and `docs/design/mobile-project-initialization.md`.

**Naming:** component/page **folders** are kebab-case (`star-button/`); **files** stay PascalCase (`StarButton.tsx`).





## Home dashboard

The **Home** tab is a live market snapshot:

- Quick chips: Markets / Pumps / Ask
- Favorites strip (when starred)
- Top movers & highest volume (Binance USDT)
- Pump radar teaser
- Pull-to-refresh; poll pauses in background

```text
modules/app/pages/home-page/
components/{molecules/section-header,quick-action-chips,organisms/dashboard-*}
libs/utils/homeDashboardQuery.ts
```

## AI assistant (Ask)

**Ask** tab chats with the multi-agent assistant via `POST /api/v1/ai/chat`:

- Device `sessionId` for multi-turn context
- Prefill from Coin detail (**Ask AI about this pair**)
- Graceful unavailable state when AI service is off
- Disclaimer: informational only — not financial advice

```bash
# backend + optional AI (see backend/.env.example AI_*)
cd backend && go run ./cmd/server
# AI on :8090 (or AI_AUTOSTART=true + AI_PYTHON)
cd mobile && npm run web
# open Ask tab
```

```text
modules/ai/pages/ai-chat-page/
components/{molecules/chat-*,organisms/chat-message-list}
libs/api/endpoints/aiApi.ts
```

## Coin detail

Tap a market row to open **Coin detail**:

- Header + 24h change
- Stats (OHLC, volume, supply)
- Interval chips + EMA toggle
- OHLCV chart (Lightweight Charts on web) + RSI pane

```text
modules/markets/pages/coin-detail-page/
components/organisms/{coin-detail-header,coin-detail-stats,interval-toolbar,candle-chart,indicator-rsi-pane}
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
- **Latest RSI** via batch indicators (`POST /indicators/batch`, multi-exchange split, ≤50 symbols)
- Local cache survives backend restarts (re-POSTs missing items on hydrate)

```text
modules/watchlist/
libs/api/endpoints/watchlistApi.ts
components/molecules/star-button
components/molecules/rsi-badge
```

Feature: `docs/features/mobile-watchlist.md` · batch RSI: `docs/features/mobile-batch-indicators.md`

## Batch RSI on lists

- **Favorites** and **Markets** rows show latest RSI from `POST /api/v1/market/indicators/batch`
- One request per exchange (max 50 symbols); partial item failures show `—`
- Poll ~45s only when focused + app active; coin detail still uses full `GET /indicators` series

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

## Pumps

The **Pumps** tab scans top quote-volume pairs for rapid moves:

- Exchange / lookback / threshold / direction filters
- Ranked hits with best return %
- Tap a hit for coin detail + event timeline
- Pull-to-refresh only (no auto-poll)

```text
modules/pumps/pages/pumps-scan-page/
libs/api/endpoints/pumpApi.ts
```

Feature: `docs/features/mobile-pumps.md`

