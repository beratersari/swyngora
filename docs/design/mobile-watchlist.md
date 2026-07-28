# Design: Mobile watchlist

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-watchlist.md`  
**Feature:** `docs/features/mobile-watchlist.md`  
**Tasks:** `project-management/tasks/mobile/watchlist/MWL-*.md`  
**Branch:** `feature/mobile-watchlist` (from latest `develop`)  
**Depends on:** Epic D (markets) + Epic E (coin detail) — both done  
**Backend:** Existing watchlist API (no backend work)  
**Parity:** `simple-frontend/` stars + watchlist chips; Telegram `/watch`

---

## 1. Problem / goal

Mobile users can browse multi-exchange markets and open coin detail, but cannot **save pairs** across sessions or jump back to a personal list. Backend already exposes a full watchlist CRUD API (in-memory, client-id tenancy).

**Goal:** Ship a mobile **Watchlist** experience that:

1. Lets users **star / unstar** pairs from Markets rows and Coin detail  
2. Shows a dedicated **Watchlist tab** with saved pairs and live-ish quotes  
3. Persists a stable **clientId** on device and syncs with `GET/POST/DELETE/PUT /api/v1/watchlist*`  
4. Uses **optimistic UI** + local merge so stars feel instant offline-to-online  
5. Stays inside **Atomic Design + View + ViewModel** (no `modules/*/components`)

Primary runtime: **Chrome** via `npm run web` (react-native-web).

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Star toggle on Markets + Coin detail | Price alerts / push |
| Watchlist tab + list screen | Auth / multi-device accounts |
| clientId persistence (local storage) | Server-side durable DB (backend still in-memory) |
| Merge local ↔ server | Paper trading |
| Enrich list with price/change (spot or ticker) | Pump scan / AI chat |
| Max-items (200) UX | Web product watchlist (separate epic if needed) |

**Note:** Backend store is **process memory**. Server restart clears remote list; local cache remains the recovery source until the user re-syncs adds. Document this in feature README and empty-state copy if needed.

---

## 3. Backend contract (source of truth)

OpenAPI: `backend/api/openapi/openapi.yaml`  
Domain: `MaxWatchlistItems = 200`; reject empty / `"default"` clientId.

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/watchlist` | Load list (`clientId` query or `X-Client-Id` header) |
| `POST` | `/api/v1/watchlist/items` | Add/upsert `{ clientId?, exchange, symbol, note? }` |
| `DELETE` | `/api/v1/watchlist/items?exchange=&symbol=&clientId=` | Remove one |
| `PUT` | `/api/v1/watchlist` | Replace full list (bulk sync / rehydrate after restart) |

**Response shape (Watchlist):**

```text
clientId, updatedAt, items[]: { exchange, symbol, note?, addedAt? }
```

**Preferred client header:** send `X-Client-Id` on all watchlist calls via `baseQuery` prepareHeaders (or per-endpoint), so DELETE/GET stay consistent without leaking id into every URL.

---

## 4. Architecture

```text
mobile/src/
├── config/
│   └── watchlistConstants.ts     # storage keys, max items (mirror 200)
├── libs/
│   ├── api/
│   │   ├── baseApi.ts            # already has tag 'Watchlist'; optional prepareHeaders
│   │   └── endpoints/
│   │       └── watchlistApi.ts   # NEW — get / add / remove / replace
│   └── utils/
│       ├── watchlistKey.ts       # watchKey(exchange, symbol)
│       ├── watchlistMerge.ts     # merge local ∪ server (local wins on conflict)
│       └── clientId.ts           # getOrCreateClientId()
├── modules/
│   └── watchlist/
│       ├── context/
│       │   └── WatchlistContext  # membership set + toggle for stars
│       ├── navigation.ts
│       ├── watchlist.types.ts
│       └── pages/
│           └── WatchlistPage/    # View + ViewModel
└── components/
    ├── molecules/
    │   └── StarButton/           # ★ toggle (presentational)
    └── organisms/
        ├── WatchlistRow/         # row for list (symbol, price, change, unstar)
        └── WatchlistList/        # FlatList wrapper
```

**Import boundaries (unchanged):** components must not import modules/libs that pull UI; RTK only under `libs/api`; ViewModels use RTK + context.

### clientId

| Rule | Detail |
|------|--------|
| Format | `mobile-<uuid-v4>` (opaque; never `default` or empty) |
| Storage | `localStorage` on web; `AsyncStorage` if native path lands later (abstract behind `libs/utils/clientId.ts`) |
| Lifecycle | Create on first watchlist access; reuse forever on that browser/device |
| Header | Prefer `X-Client-Id` on watchlist requests |

### Local cache

| Key | Content |
|-----|---------|
| `swyngora.mobile.clientId.v1` | string client id |
| `swyngora.mobile.watchlist.v1` | `{ exchange, symbol }[]` (minimal local set) |

**Merge policy (from simple-frontend):** union by `exchange|symbol`; when both present, **prefer local** so optimistic adds are not wiped by empty/lagging GET after server restart.

### WatchlistContext

Cross-screen star state without prop-drilling:

- Holds: `items`, `membership` (`Set` of watch keys), `isReady`, `toggle(exchange, symbol)`, `isWatched(exchange, symbol)`, error for max-items  
- Hydrates: localStorage → GET merge → optional PUT re-sync of missing server items (same pattern as simple-frontend)  
- Mutations: optimistic local update → RTK mutation → on failure roll back + toast/error  
- Markets + Detail pages consume context for star UI only  

Context lives under `modules/watchlist/context/` (not components).

---

## 5. UX design (phone-width)

### 5.1 Markets / detail — star control

```text
Market row:
  [★] BTCUSDT   $67,123   +1.24%
      Vol …     Mcap …

Coin detail header:
  ← BTCUSDT · binance          [★]
```

| Action | Behavior |
|--------|----------|
| Tap star (off → on) | Optimistic add; POST items; show filled star |
| Tap star (on → off) | Optimistic remove; DELETE; empty star |
| At 200 items | Block add; brief error (“Watchlist full (200)”) |
| Row press | Still opens CoinDetail (star does not steal row press — use separate hit target) |

### 5.2 Watchlist tab

New bottom tab **Watchlist** between Home and Markets (or after Markets — prefer **after Markets** so primary scan stays Markets).

```text
┌─────────────────────────────────────┐
│ Watchlist                     (n)   │
├─────────────────────────────────────┤
│ 🔍 Filter symbols…                  │  optional MWL-6
├─────────────────────────────────────┤
│ ★ BTCUSDT  binance  $…  +1.2%   [×] │
│ ★ ETH-USD  coinbase $…  −0.3%   [×] │
│ …                                   │
├─────────────────────────────────────┤
│ Empty: “Star pairs on Markets”      │
└─────────────────────────────────────┘
```

| Interaction | Behavior |
|-------------|----------|
| Row press | Navigate to CoinDetail `{ exchange, symbol }` (cross-tab: navigate into Markets stack detail, or register Detail also on watchlist stack — see §6) |
| Unstar / × | Remove from list |
| Pull-to-refresh | Refetch GET + re-enrich quotes |
| Poll | While tab focused + AppState active: re-enrich quotes ~10–15s (not full watchlist GET every tick) |

### 5.3 Empty / error / loading

| State | UX |
|-------|-----|
| First load | Skeleton list or ScreenTemplate loading |
| Empty | Illustration text + CTA “Open Markets” |
| Network error on GET | Banner; keep local stars if any |
| Mutation error | Revert optimistic; toast/inline error |

---

## 6. Navigation

```text
Main tabs:
  HomeTab
  MarketsTab   (existing stack: List, Filters, Detail)
  WatchlistTab (new stack)
    WatchlistList
    CoinDetail   ← same CoinDetailPage component, params { exchange, symbol }
```

**Why Detail on both stacks:** RN tab navigators isolate stacks; opening detail from Watchlist without a Detail route forces awkward tab jumps. Reuse `CoinDetailPage` with the same ViewModel pattern.

Update:

- `app/navigation/types.ts` — `WatchlistTab`  
- `RootNavigator.tsx` — third tab + stack  
- `modules/watchlist/navigation.ts` — screen names  

---

## 7. Data: quote enrichment

Watchlist API returns **symbols only**. For each visible item, enrich with price/change:

| Strategy | Pros | Cons |
|----------|------|------|
| **A. Per-item `GET /spot?q=SYMBOL&exchange=&limit=5`** and pick exact symbol | Simple, reuses marketApi | N requests |
| **B. Per-item `GET /ticker/24h`** | Exact pair metrics | N requests |
| **C. Group by exchange; fetch pages / batch later** | Fewer calls | More complex |

**Decision default:** **B for visible rows** with RTK cache (ticker tags already exist) + limit concurrent enrichment to on-screen items first; for v1 with typical small lists (&lt;30), fire parallel ticker queries with `skip` when list empty. Cap: if `items.length > 40`, enrich first 40 and load more on scroll (document).

Do **not** require `POST /indicators/batch` in this epic.

---

## 8. RTK surface (`watchlistApi.ts`)

```text
getWatchlist          query   GET  /api/v1/watchlist
addWatchlistItem      mutation POST /api/v1/watchlist/items
removeWatchlistItem   mutation DELETE /api/v1/watchlist/items
replaceWatchlist      mutation PUT  /api/v1/watchlist
```

- `providesTags` / `invalidatesTags`: `['Watchlist']` or `{ type: 'Watchlist', id: clientId }`  
- Export hooks from `libs/api/index.ts`  
- Types from `generated/schema` (`components['schemas']['Watchlist']`, operations)  
- `codegen:api` already includes watchlist paths — no OpenAPI change  

Optional: inject `X-Client-Id` in `baseApi` only when a module-level getter is set, **or** pass header per endpoint via `prepareHeaders` reading from a tiny `getClientIdSync()` cache after hydrate. Avoid circular imports (clientId util must not import RTK).

---

## 9. Key decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module ownership | `modules/watchlist` | Separate tab/domain; markets stay list/detail |
| Star UI location | Molecule `StarButton` + optional props on `MarketRow` / detail header | Atomic reuse |
| Tenancy | Opaque `mobile-<uuid>` in local storage | Matches backend contract; no auth yet |
| Sync model | Optimistic + local∪server merge | Proven in simple-frontend; survives backend restart |
| Navigation | Dedicated Watchlist tab + Detail on stack | Discoverable; avoids cross-tab param hacks |
| Quotes | Ticker 24h per symbol (cached) | Accurate pair stats; RTK already has endpoint |
| Max items | Surface backend 200 cap in UI | Prevent silent failures |
| Backend changes | None | API complete |
| Web `frontend/` watchlist | Out of scope | Mobile-first; board can keep separate WL-1 for web |

---

## 10. Task breakdown (MWL)

| ID | Title | Notes |
|----|-------|-------|
| MWL-1 | clientId + local storage helpers + tests | No UI |
| MWL-2 | RTK `watchlistApi` + baseApi header wiring | No page UI |
| MWL-3 | Pure merge/key helpers + unit tests | Port spirit of simple-frontend |
| MWL-4 | `WatchlistProvider` context (hydrate, toggle, membership) | |
| MWL-5 | `StarButton` + star on MarketRow + CoinDetailHeader | |
| MWL-6 | Watchlist tab navigation + WatchlistPage shell + list organisms | |
| MWL-7 | Quote enrichment + poll/refresh + empty/error UX | |
| MWL-8 | Integration tests + docs/board/changelog closeout | |

**Suggested MR grouping:**

1. **MWL-1…3** — data foundation  
2. **MWL-4…5** — stars on existing screens  
3. **MWL-6…7** — Watchlist tab + live quotes  
4. **MWL-8** — polish, docs, board  

Branch: `feature/mobile-watchlist` → MR to `develop`.

---

## 11. Testing plan

| Layer | What |
|-------|------|
| Unit | `watchKey`, `mergeWatchlists`, clientId create/reuse (mock storage) |
| Unit | Context toggle optimistic paths (mock RTK mutations) |
| Component | StarButton a11y (pressed state); MarketRow star doesn’t fire row `onPress` |
| Page | WatchlistPage ViewModel: empty list, membership map, enrichment skip |
| Manual | Chrome: star on Markets → appears on Watchlist tab → unstar → detail star in sync; restart backend → local stars remain, re-POST on hydrate |

Commands:

```bash
cd mobile && npm test && npm run lint && npm run typecheck
cd mobile && npm run web   # backend on :8080
```

---

## 12. Documentation updates (same change set as ship)

| Doc | Update |
|-----|--------|
| `docs/design/mobile-watchlist.md` | Status → Implemented |
| `docs/features/mobile-watchlist.md` | Behavior final |
| `mobile/AGENTS.md` | Watchlist module section |
| `mobile/README.md` | Tab + env notes |
| `project-management/board.md` | Tasks done |
| `CHANGELOG.md` | Added bullet under Unreleased |
| `docs/pm/mobile-epics-and-issues.md` | Epic F entry |

---

## 13. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Backend memory wipe on restart | Local cache + hydrate re-POST missing items |
| N ticker requests for large lists | Cap concurrent/visible enrichment; 200 hard max |
| Star + row press conflict | Separate touch targets; `stopPropagation` / pressable nesting careful on RN-web |
| clientId collision | UUID v4; document “device-local identity” |
| CORS + custom header | Ensure `X-Client-Id` allowed (backend CORS already allows common methods; verify `Access-Control-Allow-Headers` includes it or use query `clientId` fallback) |

**CORS check (implementer):** If OPTIONS rejects `X-Client-Id`, fall back to query param on GET/DELETE and body field on POST/PUT (API supports both). Prefer header once CORS confirmed; no backend change if query works.

---

## 14. Definition of done

- [ ] Star on Markets row and Coin detail adds/removes pair  
- [ ] Watchlist tab lists saved pairs with exchange label  
- [ ] Quotes show last price + 24h % when ticker available  
- [ ] clientId persisted; survives page reload  
- [ ] Optimistic UI; max-200 error surfaced  
- [ ] Atomic-only UI; ViewModels have no JSX  
- [ ] AppState/focus pause for quote polling  
- [ ] Tests green; docs/board/changelog updated  

**Last updated:** 2026-07-27
