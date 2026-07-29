# MCAT-4: Apply tag to Markets list (context + navigation)

| Field | Value |
|---|---|
| **ID** | MCAT-4 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-4.md` |

## Summary

Wire discovery selection to the existing Markets list:

1. On tag select: `applyFilters` / context API sets `selectedTags` to **`[tag]`** (replace multi), keep or set sensible quote/sort defaults, reset pagination offset.  
2. Navigate to Markets list screen (pop Categories or `navigate` to list).  
3. Confirm `MarketsPage` already passes `tags` into spot RTK query (it should via context); fix any gap.  
4. Markets toolbar: show active category chip / clear affordance when `selectedTags.length === 1` (or >0).  
5. Entry from Markets to Categories route (toolbar button / chip).

No second list page unless Markets integration is blocked — prefer reuse.

## Design

`docs/design/mobile-category-discovery.md` §4, §7

## Acceptance

- [x] Selecting Meme shows only tag-filtered spot rows (manual or test with mocked API)  
- [x] Clearing tag restores unfiltered list behavior  
- [x] Pagination / poll still work under tag filter  
- [x] Context tests or page tests cover apply + navigate  
- [x] Status → done when finished  

## Status

`done`
