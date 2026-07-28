# MHOME-5: Empty / partial failure / loading UX

| Field | Value |
|---|---|
| **ID** | MHOME-5 |
| **Epic** | mobile-home-dashboard |
| **Status** | done |
| **Area** | mobile |
| **Depends on** | MHOME-4 |
| **Path** | `project-management/tasks/mobile/home/MHOME-5.md` |

## Summary

Production-ready section UX:

| Case | UX |
|------|-----|
| Initial load | Skeletons per section |
| One section fails | Banner/error on that section only; others remain |
| Empty favorites | CTA “Open Markets” / star hint |
| Empty pumps | “No hits” + link to Pumps filters |
| Offline / backend down | Clear retry on failing sections |
| Background | Poll paused caption optional |

Pump / indicators disclaimers where relevant.

## Acceptance

- [x] Partial failure does not blank entire Home  
- [x] Retry hooks wired  
- [x] Status updated  

## Status

`done`
