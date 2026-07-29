# MCAT-5: Home featured strip + Markets entry points

| Field | Value |
|---|---|
| **ID** | MCAT-5 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-5.md` |

## Summary

Surface category discovery outside the filter form:

**Home**

- Section: featured category chips (MCAT-2 organism)  
- Data: `useListProductTagsQuery` + MCAT-1 intersect (or shared home helper)  
- Tap chip → apply Markets tag + navigate Markets tab  
- “See all” → Markets stack `Categories` route (ensure nested navigate works)

**Markets**

- Toolbar / chrome entry to Categories (if not fully done in MCAT-4)  
- Optional: empty-state hint when no filters: “Browse categories”

Reuse AppState pause patterns; tags query does not need fast polling.

## Design

`docs/design/mobile-category-discovery.md` §4–5

## Acceptance

- [x] Home shows featured chips when tags succeed  
- [x] Home partial failure: hide strip or show compact error without breaking movers/volume  
- [x] Chip navigation lands on filtered Markets list  
- [x] “See all” opens Categories page  
- [x] Home tests updated  
- [x] Status → done when finished  

## Status

`done`
