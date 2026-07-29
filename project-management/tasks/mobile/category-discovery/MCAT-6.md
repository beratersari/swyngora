# MCAT-6: Loading / empty / error / i18n

| Field | Value |
|---|---|
| **ID** | MCAT-6 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-6.md` |

## Summary

Product polish for category discovery:

| State | Behavior |
|-------|----------|
| Tags loading | Skeleton / chips skeleton on Categories + Home strip |
| Tags error | Message + retry |
| Tags empty | Explain catalog unavailable |
| Markets empty under tag | Empty list copy: no markets for this category |
| i18n | en + tr keys (`markets` and/or `home`); no hard-coded user copy in View |

Also: ensure active tag is visible on Markets so users know why the list is filtered.

## Design

`docs/design/mobile-category-discovery.md` §8

## Acceptance

- [x] en/tr parity for new strings  
- [x] Empty + error paths covered in tests or documented manual checks  
- [x] No English-only hard-coded UI strings on new surfaces  
- [x] Status → done when finished  

## Status

`done`
