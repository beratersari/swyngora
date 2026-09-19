# MCAT-2: Atomic category UI (grid / strip)

| Field | Value |
|---|---|
| **ID** | MCAT-2 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-2.md` |

## Summary

Add Atomic Design UI under `mobile/src/components/` (kebab-case folders). Prefer composing existing `chip` / `chip-group` / `section-header` before inventing duplicates.

Suggested:

| Folder | Level | Role |
|--------|-------|------|
| `organisms/category-chip-grid` | organism | Searchable/all + optional featured block; props only |
| `organisms/category-section` (or molecule strip) | organism | Home horizontal featured chips + “See all” |

Files: `*.tsx`, `*.types.ts`, `*.styles.ts`, `index.ts`, tests.  
No RTK or navigation inside atoms/molecules/organisms.

## Design

`docs/design/mobile-category-discovery.md` §5

## Acceptance

- [x] Loading skeleton / empty props supported  
- [x] Multi vs single selection: discovery uses **single** select callback `onSelectTag(tag)`  
- [x] Accessible labels / press targets suitable for web + RN  
- [x] Unit/render tests for primary organism(s)  
- [x] Status → done when finished  

## Status

`done`
