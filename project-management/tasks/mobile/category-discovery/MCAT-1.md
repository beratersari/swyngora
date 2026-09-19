# MCAT-1: Constants + category query / label helpers + tests

| Field | Value |
|---|---|
| **ID** | MCAT-1 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-1.md` |

## Summary

Pure helpers + constants (no UI):

- `config/categoryConstants.ts` — featured tag ids (e.g. Meme, AI, defi, Layer1_Layer2, Payments), default tags exchange (`binance`), optional max featured count  
- `libs/utils/categoryQuery.ts` (or extend `spotQuery`) —  
  - `intersectFeaturedTags(liveTags, featured)`  
  - `filterTagsBySearch(tags, query)`  
  - `buildCategorySpotParams({ tag, exchange, quote, … })` → stable RTK/spot args with `tag`  
  - optional humanize label helper for `Layer1_Layer2` → display string (i18n keys preferred; pure fallback OK)  
- Unit tests for all pure functions  
- Export from `libs/utils/index.ts`

## Design

`docs/design/mobile-category-discovery.md` §3–4 · `MCAT-A.md`

## Acceptance

- [x] Featured ∩ live is order-stable (featured order preserved)  
- [x] Search is case-insensitive substring  
- [x] Spot params omit empty tag; single tag only for discovery path  
- [x] No React / navigation imports  
- [x] Tests green  
- [x] Status → done when finished  

## Status

`done`
