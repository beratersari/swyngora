# MHOME-3: HomePage ViewModel

| Field | Value |
|---|---|
| **ID** | MHOME-3 |
| **Epic** | mobile-home-dashboard |
| **Status** | todo |
| **Area** | mobile |
| **Depends on** | MHOME-1 |
| **Path** | `project-management/tasks/mobile/home/MHOME-3.md` |

## Summary

Expand `modules/app/pages/home-page/HomePage.viewModel.ts`:

- Parallel RTK: movers spot, volume spot, pump scan (skip when unfocused)
- Favorites: read `useOptionalWatchlist` / context; optional batch RSI for strip
- `useAppStateActive` + `useIsFocused` → poll interval 0 when inactive
- Map errors per section (partial failure OK)
- Actions: open Markets, Pumps, Ask, detail navigation payloads

Keep health as optional footer, not the main content.

## Design

`docs/design/mobile-home-dashboard.md` §6

## Acceptance

- [ ] Section-level loading/error fields on VM  
- [ ] No JSX in ViewModel  
- [ ] Status updated  

## Status

`todo`
