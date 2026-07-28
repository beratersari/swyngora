# MHOME-2: Atomic dashboard UI

| Field | Value |
|---|---|
| **ID** | MHOME-2 |
| **Epic** | mobile-home-dashboard |
| **Status** | todo |
| **Area** | mobile |
| **Depends on** | MHOME-1 types (can stub props) |
| **Path** | `project-management/tasks/mobile/home/MHOME-2.md` |

## Summary

Presentational components under `src/components/` (kebab-case):

| Level | Component | Role |
|-------|-----------|------|
| Molecule | `section-header` | Title + optional action (“See all”) |
| Molecule / organism | `dashboard-market-row` | Symbol, price, change%, optional RSI |
| Organism | `dashboard-section-list` | Title + list of rows / empty |
| Organism | `pump-teaser-card` | Compact pump hit preview |
| Molecule | `quick-action-chips` | Markets / Pumps / Ask shortcuts |

Props only — no RTK. File split: types, styles, index.

## Design

`docs/design/mobile-home-dashboard.md` §5

## Acceptance

- [ ] Atomic only; no module components  
- [ ] Brand tokens / existing Text, Chip patterns  
- [ ] Status updated  

## Status

`todo`
