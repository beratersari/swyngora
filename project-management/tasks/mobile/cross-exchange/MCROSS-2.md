# MCROSS-2: Atomic compare UI (row + list organism)

| Field | Value |
|---|---|
| **ID** | MCROSS-2 |
| **Epic** | mobile-cross-exchange-compare |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/cross-exchange/MCROSS-2.md` |

## Summary

Add Atomic UI under `mobile/src/components/` (kebab-case):

| Folder | Level | Role |
|--------|-------|------|
| `organisms/cross-exchange-compare` | organism | Section title, list of rows, disclaimer, loading skeletons |
| optional `molecules/cross-exchange-row` | molecule | Single venue metrics line |

Files: `*.tsx`, `*.types.ts`, `*.styles.ts`, `index.ts`, tests.  
Props only — no RTK/navigation inside.

Row props: exchange label, symbol, price, change, volume, status, isSource, onPress.

## Design

`docs/design/mobile-cross-exchange-compare.md` §6

## Acceptance

- [x] Loading / unavailable / error row states  
- [x] Source venue visual distinction  
- [x] Press invokes callback with exchange+symbol  
- [x] Render tests  
- [x] Status → done when finished  

## Status

`done`
