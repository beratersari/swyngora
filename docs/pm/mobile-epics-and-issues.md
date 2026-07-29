# GitLab project management: Mobile epics & issues

**Project:** `trace-analysis/swyngora`  
**Host:** https://nova.teachx.ai  
**Labels (create if missing):** `mobile`, `type::epic`, `type::feature`, `type::chore`, `priority::p0`, `priority::p1`

**Local board (current):** [`project-management/`](../../project-management/) — day-to-day status until GitLab issues exist.

---

## Epic C — Mobile project initialization (FIRST for mobile)

| Field | Value |
|---|---|
| **Title** | `[mobile] Project initialization` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p0` |
| **Design plan** | `docs/design/mobile-project-initialization.md` |
| **System design** | `docs/design/mobile-system-design.md` |
| **Decision** | `project-management/decisions/002-react-native-cli-modules-viewmodel.md` |

### Epic description (paste into GitLab)

```markdown
## Summary

Initialize the production React Native app under `mobile/` (no Expo) so feature work can start.

This is **P0 for the mobile track** and **blocks** multi-exchange spot markets UI on mobile.

## Goals

- React Native CLI + TypeScript scaffold (**no Expo**)
- Lint / test tooling + path aliases
- Atomic Design under components (no pages) + modules that own pages
- View + ViewModel page contract
- Redux Toolkit + RTK Query baseApi
- OpenAPI codegen from backend/api/openapi/openapi.yaml
- React Navigation shell + AppState polling hygiene
- **Same brand color tokens as frontend**
- Package README + AGENTS.md

## Out of scope

Full markets table, coin detail, charts, auth, Expo.

## Acceptance

- npm run android launches shell (documented host)
- npm run codegen:api works
- colors.ts brand hex matches frontend
- Home + MarketsList stubs with ViewModels
- Structure matches mobile/AGENTS.md

## Child issues

See MINIT-1 … MINIT-9.
```

### Child issues

| ID | Title |
|---|---|
| MINIT-1 | Scaffold React Native CLI TypeScript app (no Expo) |
| MINIT-2 | Lint, format, Jest, path aliases |
| MINIT-3 | libs + Atomic + modules skeleton + boundary ESLint |
| MINIT-4 | libs/api store + RTK baseApi + env |
| MINIT-5 | OpenAPI codegen → libs/api/generated |
| MINIT-6 | Navigation shell + AppState + providers |
| MINIT-7 | Color tokens (match frontend) + core atoms + ScreenTemplate |
| MINIT-8 | Home + Markets stub pages with ViewModels |
| MINIT-9 | Package docs + root links + changelog |

Full task text: `project-management/tasks/mobile/init/MINIT-*.md`.


---

## Epic D — Mobile multi-exchange spot markets / dashboard

| Field | Value |
|---|---|
| **Title** | `[mobile] Multi-exchange spot markets dashboard` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p0` |
| **Design** | `docs/design/mobile-markets-dashboard.md` |
| **Branch** | `feature/mobile-spot-markets` |

### Child issues

| ID | Title |
|---|---|
| MMKT-1 | RTK marketApi: exchanges, tags, spot |
| MMKT-2 | Formatters + spot query helpers |
| MMKT-3 | ExchangeChips + MarketsFilterBar |
| MMKT-4 | MarketRow + MarketsList |
| MMKT-5 | MarketsPage ViewModel |
| MMKT-6 | MarketsPage View + pull-to-refresh |
| MMKT-7 | Empty/error/loading UX + tests |
| MMKT-8 | Docs + board closeout |

Full task text: `project-management/tasks/mobile/markets/MMKT-*.md`.


---

## Epic E — Mobile coin detail + indicators

| Field | Value |
|---|---|
| **Title** | `[mobile] Coin detail + indicators` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-coin-detail.md` |
| **Tasks** | `project-management/tasks/mobile/detail/MDET-*.md` |

### Child issues

| ID | Title |
|---|---|
| MDET-1 | RTK detail endpoints |
| MDET-2 | Navigation CoinDetail + row press |
| MDET-3 | CoinDetailPage shell + header/stats |
| MDET-4 | Interval toolbar + candle chart |
| MDET-5 | RSI/EMA organisms |
| MDET-6 | Loading/error/polling tests |
| MDET-7 | Docs closeout |

Full task text: `project-management/tasks/mobile/detail/MDET-*.md`.


---

## Epic F — Mobile watchlist

| Field | Value |
|---|---|
| **Title** | `[mobile] Watchlist` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p0` |
| **Design** | `docs/design/mobile-watchlist.md` |
| **Feature** | `docs/features/mobile-watchlist.md` |
| **Epic file** | `project-management/epics/mobile-watchlist.md` |
| **Tasks** | `project-management/tasks/mobile/watchlist/MWL-*.md` |
| **Branch** | `feature/mobile-watchlist` |

### Epic description (paste into GitLab)

```markdown
## Summary

Mobile watchlist: star/unstar pairs from Markets and Coin detail, dedicated Watchlist tab,
device-local clientId sync to existing GET/POST/DELETE/PUT `/api/v1/watchlist*`, ticker enrichment.

## Goals

- Star on market rows + coin detail
- Watchlist tab with list + navigate to detail
- Optimistic local ∪ server merge (simple-frontend parity)
- Max 200 items UX; no new backend work

## Out of scope

Auth, alerts, web frontend watchlist, durable server store, pumps/AI.

## Child issues

MWL-1 … MWL-8
```

### Child issues

| ID | Title |
|---|---|
| MWL-1 | clientId + local storage helpers |
| MWL-2 | RTK watchlistApi endpoints |
| MWL-3 | watchKey + merge pure helpers |
| MWL-4 | WatchlistProvider context |
| MWL-5 | StarButton + Markets/Detail wiring |
| MWL-6 | Watchlist tab + WatchlistPage |
| MWL-7 | Quote enrichment + poll/empty/error |
| MWL-8 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/watchlist/MWL-*.md`.


---

## Epic G — Mobile pump / dump radar

| Field | Value |
|---|---|
| **Title** | `[mobile] Pump / dump radar` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-pumps.md` |
| **Feature** | `docs/features/mobile-pumps.md` |
| **Analysis** | `project-management/tasks/mobile/pumps/MPUMP-A.md` |
| **Epic file** | `project-management/epics/mobile-pumps.md` |
| **Tasks** | `project-management/tasks/mobile/pumps/MPUMP-*.md` |
| **Branch** | `feature/mobile-pumps` |

### Epic description (paste into GitLab)

```markdown
## Summary

Mobile pump/dump radar: scan top-volume symbols via GET /api/v1/market/pumps/scan,
list ranked hits, open coin detail, show per-symbol events via GET /pumps.
Backend already implemented; MCP tools exist. No new backend work.

## Goals

- Pumps tab with scan list + filters (exchange, threshold, lookback, direction)
- Hit → coin detail
- Detail section: pump/dump event timeline
- Informational disclaimer; no aggressive polling

## Out of scope

Alerts, trading, AI narratives, web frontend pumps UI.

## Child issues

MPUMP-A (analysis done) · MPUMP-1 … MPUMP-7
```

### Child issues

| ID | Title |
|---|---|
| MPUMP-A | API field matrix analysis |
| MPUMP-1 | RTK pumpApi endpoints |
| MPUMP-2 | formatPump + pumpQuery helpers |
| MPUMP-3 | Pump Atomic UI |
| MPUMP-4 | PumpsScanPage + tab navigation |
| MPUMP-5 | Coin detail pump events section |
| MPUMP-6 | Loading / empty / error / disclaimer |
| MPUMP-7 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/pumps/MPUMP-*.md`.


---

## Epic H — Mobile batch indicators

| Field | Value |
|---|---|
| **Title** | `[mobile] Batch indicators (list RSI)` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-batch-indicators.md` |
| **Feature** | `docs/features/mobile-batch-indicators.md` |
| **Analysis** | `project-management/tasks/mobile/batch-indicators/MBIND-A.md` |
| **Epic file** | `project-management/epics/mobile-batch-indicators.md` |
| **Tasks** | `project-management/tasks/mobile/batch-indicators/MBIND-*.md` |
| **Branch** | `feature/mobile-batch-indicators` |

### Epic description (paste into GitLab)

```markdown
## Summary

Enrich Favorites (P0) and Markets list (P1) with latest RSI via
POST /api/v1/market/indicators/batch. Avoid N+1 GET /indicators.
Coin detail series unchanged. No new backend work.

## Goals

- RTK batch endpoint + helpers (chunk ≤50, multi-exchange split)
- RSI badge on Favorites (and Markets rows)
- Partial failure UX + AppState pause + disclaimer

## Out of scope

Alerts, trading, editable periods on lists, web frontend columns.

## Child issues

MBIND-A (analysis) · MBIND-1 … MBIND-7
```

### Child issues

| ID | Title |
|---|---|
| MBIND-A | API field matrix analysis |
| MBIND-1 | RTK postIndicatorsBatch endpoint |
| MBIND-2 | Chunk/group/format helpers |
| MBIND-3 | RSI badge + row props |
| MBIND-4 | Favorites batch enrichment (P0) |
| MBIND-5 | Markets list batch enrichment (P1) |
| MBIND-6 | Loading / partial failure / disclaimer |
| MBIND-7 | Docs + board + changelog closeout |


---

## Epic I — Mobile AI assistant chat

| Field | Value |
|---|---|
| **Title** | `[mobile] AI assistant chat` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-ai-chat.md` |
| **Feature** | `docs/features/mobile-ai-chat.md` |
| **Analysis** | `project-management/tasks/mobile/ai-chat/MAI-A.md` |
| **Epic file** | `project-management/epics/mobile-ai-chat.md` |
| **Tasks** | `project-management/tasks/mobile/ai-chat/MAI-*.md` |
| **Branch** | `feature/mobile-ai-chat` |

### Epic description (paste into GitLab)

```markdown
## Summary

Mobile Ask tab: multi-turn chat via POST /api/v1/ai/chat (Go proxy to Python multi-agent).
OpenAPI + codegen first; RTK mutation; Atomic chat UI; context from coin detail.

## Goals

- Document AI chat in OpenAPI
- Ask tab with sessionId multi-turn
- Context prefill from detail
- Graceful 503 when AI offline
- Not financial advice disclaimer

## Out of scope

Streaming UI, auth/history store, web chat, new LLM vendors.

## Child issues

MAI-A · MAI-1 … MAI-8
```

### Child issues

| ID | Title |
|---|---|
| MAI-A | API field matrix analysis |
| MAI-1 | OpenAPI for AI chat + client codegen |
| MAI-2 | RTK aiApi chat mutation |
| MAI-3 | sessionId + message model helpers |
| MAI-4 | Atomic chat UI |
| MAI-5 | Ask tab + AiChatPage ViewModel |
| MAI-6 | Context chips from Markets / Detail / Pumps |
| MAI-7 | Loading / 503 / error / disclaimer |
| MAI-8 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/ai-chat/MAI-*.md`.


---

## Epic J — Mobile home dashboard

| Field | Value |
|---|---|
| **Title** | `[mobile] Home market dashboard` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-home-dashboard.md` |
| **Feature** | `docs/features/mobile-home-dashboard.md` |
| **Epic file** | `project-management/epics/mobile-home-dashboard.md` |
| **Tasks** | `project-management/tasks/mobile/home/MHOME-*.md` |
| **Branch** | `feature/mobile-home-dashboard` |

### Epic description (paste into GitLab)

```markdown
## Summary

Replace mobile Home scaffold with live dashboard: favorites, top movers,
high volume, pump teaser, quick links to Markets / Pumps / Ask.
Uses existing spot / pumps / watchlist APIs only.

## Goals

- Parallel RTK widgets with AppState pause
- Deep links into existing stacks
- Partial failure UX per section

## Out of scope

New backend, alerts, layout editor, web home.

## Child issues

MHOME-1 … MHOME-7
```

### Child issues

| ID | Title |
|---|---|
| MHOME-1 | Dashboard constants + query helpers |
| MHOME-2 | Atomic dashboard UI |
| MHOME-3 | HomePage ViewModel |
| MHOME-4 | HomePage View + deep links |
| MHOME-5 | Empty / partial failure / loading UX |
| MHOME-6 | Tests |
| MHOME-7 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/home/MHOME-*.md`.


---

## Epic K — Mobile category discovery

| Field | Value |
|---|---|
| **Title** | `[mobile] Category discovery` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-category-discovery.md` |
| **Feature** | `docs/features/mobile-category-discovery.md` |
| **Analysis** | `project-management/tasks/mobile/category-discovery/MCAT-A.md` |
| **Epic file** | `project-management/epics/mobile-category-discovery.md` |
| **Tasks** | `project-management/tasks/mobile/category-discovery/MCAT-*.md` |
| **Branch** | `feature/mobile-category-discovery` |

### Epic description (paste into GitLab)

```markdown
## Summary

Mobile category discovery: browse Binance product-catalog tags (Meme, AI, defi, …)
via GET /api/v1/market/tags, open tag-filtered markets via GET /spot?tag=.
Home featured chips + Categories browse page; reuse Markets list/context.
No new backend work.

## Goals

- Featured categories on Home
- Categories browse (search + grid) on Markets stack
- Single-tag apply → filtered Markets list
- Empty / error / i18n (en/tr)

## Out of scope

New endpoints, multi-tag discovery UX, alerts, web frontend category UI.

## Child issues

MCAT-A · MCAT-1 … MCAT-7
```

### Child issues

| ID | Title |
|---|---|
| MCAT-A | Tags + spot tag-filter field matrix |
| MCAT-1 | Constants + category query helpers |
| MCAT-2 | Atomic category UI |
| MCAT-3 | Categories browse page + route |
| MCAT-4 | Apply tag to Markets list |
| MCAT-5 | Home featured strip + Markets entry |
| MCAT-6 | Loading / empty / error / i18n |
| MCAT-7 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/category-discovery/MCAT-*.md`.


---

## Epic L — Mobile cross-exchange coin comparison

| Field | Value |
|---|---|
| **Title** | `[mobile] Cross-exchange coin comparison` |
| **Type** | Epic |
| **Labels** | `mobile`, `priority::p1` |
| **Design** | `docs/design/mobile-cross-exchange-compare.md` |
| **Feature** | `docs/features/mobile-cross-exchange-compare.md` |
| **Analysis** | `project-management/tasks/mobile/cross-exchange/MCROSS-A.md` |
| **Epic file** | `project-management/epics/mobile-cross-exchange-compare.md` |
| **Tasks** | `project-management/tasks/mobile/cross-exchange/MCROSS-*.md` |
| **Branch** | `feature/mobile-cross-exchange-compare` |

### Epic description (paste into GitLab)

```markdown
## Summary

Coin detail “Across exchanges” section: parallel GET /ticker/24h for
binance / coinbase / bybit with client symbol mapping (BTCUSDT ↔ BTC-USD).
No new backend endpoint in v1. Partial failure per venue; tap opens that venue detail.

## Goals

- Symbol mapping helpers + tests
- Atomic compare UI
- Wire into coin detail ViewModel + page
- Poll pause / i18n / disclaimer

## Out of scope

New compare API, FX conversion, alerts, web UI.

## Child issues

MCROSS-A · MCROSS-1 … MCROSS-7
```

### Child issues

| ID | Title |
|---|---|
| MCROSS-A | Field matrix + symbol mapping analysis |
| MCROSS-1 | Constants + cross-exchange helpers |
| MCROSS-2 | Atomic compare UI |
| MCROSS-3 | Coin detail ViewModel parallel tickers |
| MCROSS-4 | Wire section + navigate to venue |
| MCROSS-5 | Loading / partial failure / i18n |
| MCROSS-6 | Tests polish |
| MCROSS-7 | Docs + board + changelog closeout |

Full task text: `project-management/tasks/mobile/cross-exchange/MCROSS-*.md`.
