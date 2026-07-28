# MHOME-4: HomePage View + deep links

| Field | Value |
|---|---|
| **ID** | MHOME-4 |
| **Epic** | mobile-home-dashboard |
| **Status** | todo |
| **Area** | mobile |
| **Depends on** | MHOME-2, MHOME-3 |
| **Path** | `project-management/tasks/mobile/home/MHOME-4.md` |

## Summary

Rebuild `HomePage.tsx` composition:

- ScrollView + pull-to-refresh (`onRefresh` refetches all sections)
- Sections: Favorites · Movers · Volume · Pumps · Quick actions · Language switcher (footer)
- Navigation:
  - Market row → Markets stack detail (or shared detail route pattern)
  - Pump teaser → Pumps tab / detail
  - Chips → MarketsTab / PumpsTab / AskTab
- Preserve injectable `viewModel` for tests

## Design

`docs/design/mobile-home-dashboard.md` §5–6

## Acceptance

- [ ] Deep links work from Home  
- [ ] Pull-to-refresh resets sections  
- [ ] Status updated  

## Status

`todo`
