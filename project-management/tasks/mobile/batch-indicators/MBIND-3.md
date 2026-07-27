# MBIND-3: Atomic RSI badge + row prop extensions

| Field | Value |
|---|---|
| **ID** | MBIND-3 |
| **Epic** | mobile-batch-indicators |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MBIND-2 |

## Summary

Presentational UI only:

- Molecule **`rsi-badge`** (or extend existing chip): shows `RSI 62.4` + tone color; loading skeleton state; unavailable `—`
- Extend **`watchlist-row`** and **`market-row`** view models/props with optional:
  - `rsiLabel`, `rsiTone`, `rsiLoading`
- Keep organisms pure (no RTK)
- Colocated tests for badge + row rendering with/without RSI

## Design

Atomic Design · kebab-case folders · design § UI

## Acceptance

- [ ] Badge handles number / null / loading  
- [ ] Rows optional RSI without breaking existing tests  
- [ ] No network imports in components  
- [x] Status → done  

## Status

`done`
