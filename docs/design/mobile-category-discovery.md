# Design: Mobile category discovery

**Status:** Implemented  
**Epic:** `project-management/epics/mobile-category-discovery.md`  
**Feature:** `docs/features/mobile-category-discovery.md`  
**Tasks:** `project-management/tasks/mobile/category-discovery/MCAT-*.md`  
**Branch:** `feature/mobile-category-discovery`  
**Depends on:** Markets dashboard + Home (done)  
**Backend:** Existing tags + spot tag filter only — **no new endpoints**

---

## 1. Problem / goal

Product tags (Meme, defi, AI, Layer1_Layer2, …) already power **Markets filter**, but discovery is buried: users must open Markets → Filters → multi-select tags. Mobile needs a **browse-first** path: pick a category, see matching markets.

**Goal:** Category discovery as a first-class mobile flow, reusing `/tags` + `/spot?tag=`.

---

## 2. Product framing

| In this epic | Out of scope |
|--------------|--------------|
| Featured categories + full tag browse | New backend / MCP tools (tags already exist) |
| Tag → filtered markets list | Alerts, paper trading |
| Home category strip | Custom user category taxonomy |
| Reuse Markets list + context | Web (`frontend/`) redesign |
| Binance catalog as source of tags | Inventing tags for non-crypto |

---

## 3. Data sources

| Need | Endpoint | Notes |
|------|----------|-------|
| Category list | `GET /api/v1/market/tags?exchange=` | Unique catalog tags; **non-empty for Binance**. Other venues return `[]` — still use Binance tags for discovery labels; spot rows on Coinbase/Bybit can inherit tags by base (server enrichment). |
| Markets in category | `GET /api/v1/market/spot` with `tag` or `tags` | OR match if multiple; v1 UI selects **one** primary tag at a time |
| Sort / quote | Same spot params | Default: `quote=USDT`, `sort=quoteVolume`, `order=desc` |

**Featured set (constants):** curated subset for Home / quick chips (e.g. `Meme`, `AI`, `defi`, `Layer1_Layer2`, `Payments`) — only show chips that exist in the live `/tags` response.

---

## 4. UX flow

```text
Home — featured category chips
  → tap "Meme"
  → Markets stack with selectedTags=["Meme"] (list filtered)
  → tap row → Coin detail

OR

Markets — "Categories" entry (toolbar / chip)
  → Categories browse page (search + grid)
  → tap tag
  → Markets list with that tag applied
```

### Rules

- Selecting a category **replaces** prior multi-tag filter with that single tag (clear other tags for predictable results). User can still open Filters to refine.
- Exchange: discovery defaults to **binance** when loading tags. If Markets is already on Coinbase/Bybit, still request tags from binance for the catalog list; spot query uses **current Markets exchange** with the same tag string (server enriches tags by base where possible). Document empty-list UX when zero matches.
- Pagination / poll / AppState: reuse MarketsPage behavior when viewing filtered list.
- No aggressive polling on the category grid itself (tags change slowly; cache-friendly).

---

## 5. UI structure

```text
Categories browse (Markets stack)
  [Search tags]
  Featured section (chips)
  All tags (scrollable chip grid / tiles)
  Empty / error states

Home
  … existing widgets …
  Categories strip (featured chips + “See all”)

Markets toolbar
  Entry to Categories (icon/chip) when no tag; active tag chip when filtered
```

Atomic (kebab-case under `components/`):

| Component | Level | Role |
|-----------|-------|------|
| `category-chip` or reuse `chip` / `chip-group` | molecule | Single tag select |
| `category-chip-grid` | organism | Featured + all tags layout |
| optional `category-section` | organism | Home strip (header + horizontal chips) |

**Pages:** prefer `modules/markets/pages/categories-page/` (browse only). Filtered results stay on **MarketsPage** via context — avoid duplicating list/pagination.

---

## 6. ViewModel sketch

```ts
// CategoriesPage ViewModel
{
  featuredTags: string[];      // curated ∩ live tags
  allTags: string[];           // from listProductTags, filtered by search
  search: string;
  isLoading: boolean;
  errorMessage: string | null;
  onSearchChange(q: string): void;
  onSelectTag(tag: string): void; // applyFilters({ selectedTags: [tag] }) + navigate Markets
  onRetry(): void;
}
```

Home: thin wiring — featured tags query + navigate/apply filter.

---

## 7. Navigation

| Route | Stack | Params |
|-------|-------|--------|
| `Categories` | Markets (or Home stack deep-link into Markets) | none |
| Existing `Markets` list | Markets | uses context `selectedTags` |

Deep link from Home: set Markets context tags then `navigate('MarketsTab')` (same pattern as Home → Markets).

---

## 8. i18n

Namespace `markets` (or small `categories` if copy grows): titles, search placeholder, empty tags, empty markets for tag, featured header, “See all categories”.

Locales: **en** + **tr** parity.

---

## 9. Testing

- Helpers: featured ∩ live, tag query param builder, search filter  
- CategoriesPage: loading / empty / select applies context  
- Home strip: chips render when tags load; tap navigates  
- No live network in unit tests  

---

## 10. Acceptance (epic-level)

- [ ] User can browse categories without opening the full filter form first  
- [ ] Selecting a tag shows spot markets for that tag  
- [ ] Home shows featured category entry points  
- [ ] Empty/error states for tags and zero-result lists  
- [ ] Atomic Design + ViewModel; RTK only under `libs/api`  
- [ ] Tests + docs closed  

## 11. Out of scope

- New OpenAPI paths  
- Multi-tag browse mode (OR chips on category page) — Filters remains multi-select  
- Auth, alerts, web UI  
- Persisting last category across app reinstall (optional later via storage)
