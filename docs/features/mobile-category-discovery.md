# Feature: Category discovery (mobile)

**Status:** Implemented  
**Surface:** Product mobile (`mobile/`) — Chrome via react-native-web  
**Backend:** Existing product tags + spot tag filter (no new endpoints)  
**Epic:** `project-management/epics/mobile-category-discovery.md`  
**Design:** `docs/design/mobile-category-discovery.md`  
**Tasks:** `project-management/tasks/mobile/category-discovery/MCAT-*.md`

---

## 1. Problem / goal

Let users **discover markets by theme** (Meme, AI, defi, Layer1, …) using Binance product-catalog tags, instead of only hunting via Markets filters.

---

## 2. Behavior (happy path)

1. Open **Home** → tap a featured category chip (e.g. Meme).  
2. Land on **Markets** list filtered to that tag.  
3. Or open **Markets → Categories**, search/browse all tags, pick one.  
4. Tap a row → coin detail (existing).  
5. Clear filter via Markets filters / clear chip to see full list again.

### Limits

- Tag catalog is Binance marketing tags (non-crypto tags already excluded server-side).  
- Multi-tag OR filters stay on the existing Markets filter form; discovery v1 is **single-tag**.  
- Informational market data only — not financial advice.

---

## 3. APIs

| Method | Path | Use |
|--------|------|-----|
| `GET` | `/api/v1/market/tags` | Category list |
| `GET` | `/api/v1/market/spot` | `tag` / `tags` filter + sort/pagination |

MCP already exposes spot list (tag filter) and does not need a dedicated “list tags” tool for this UI-only epic; no new MCP requirement unless agents later need tag enumeration as a first-class tool (optional follow-up).

---

## 4. Code homes (planned)

| Area | Path |
|------|------|
| Browse page | `mobile/src/modules/markets/pages/categories-page/` |
| List results | Existing `markets-page` + `MarketsContext.selectedTags` |
| Helpers | `mobile/src/libs/utils/categoryQuery.ts` (or similar) |
| Constants | `mobile/src/config/categoryConstants.ts` |
| Organisms | `mobile/src/components/organisms/category-chip-grid/` etc. |

---

## 5. How to verify

```bash
cd backend && go run ./cmd/server
cd mobile && npm run web
# Home → category chip → Markets filtered
# Markets → Categories → pick tag → list updates
```

---

## 6. Known limitations / follow-ups

- Empty results possible on non-Binance venues for niche tags.  
- Featured set is curated in constants, not server-driven.  
- Optional later: pin last category, multi-select discovery, tag counts.
