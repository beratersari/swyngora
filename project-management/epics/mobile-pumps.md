# Epic G: Mobile pump / dump radar

**Priority:** P1 (mobile track — after watchlist)  
**Status:** done  
**Depends on:** Markets + coin detail (done)  
**Branch:** `feature/mobile-pumps`  
**Design plan:** `docs/design/mobile-pumps.md`  
**Feature:** `docs/features/mobile-pumps.md`  
**Analysis:** `project-management/tasks/mobile/pumps/MPUMP-A.md`  
**Tasks folder:** `project-management/tasks/mobile/pumps/`

## Goal

Let mobile users **discover rapid price moves** via multi-symbol pump scan and inspect **per-coin pump/dump events** on detail — using existing backend APIs. Atomic Design (kebab-case folders) + View + ViewModel; primary runtime **Chrome**.

## APIs (existing backend)

| UI | Path |
|----|------|
| Scan radar | `GET /api/v1/market/pumps/scan` |
| Symbol events | `GET /api/v1/market/pumps` |
| Intervals / exchanges | existing market endpoints |

MCP tools already exist: `detect_pump_events`, `scan_pump_events`.

## Tasks

- [x] MPUMP-A — API field matrix analysis  
- [x] MPUMP-1 — RTK pumpApi  
- [x] MPUMP-2 — format + query helpers + tests  
- [x] MPUMP-3 — Atomic organisms (filters, hit list, event list)  
- [x] MPUMP-4 — PumpsScanPage + tab navigation  
- [x] MPUMP-5 — Coin detail pumps section  
- [x] MPUMP-6 — Loading / empty / error / disclaimer  
- [x] MPUMP-7 — Tests polish + docs/board/changelog  

## Acceptance

- Pumps tab shows scan hits with return %  
- Filters refetch scan  
- Hit opens coin detail  
- Detail lists pump events  
- Disclaimer visible  
- No aggressive polling  
- Tests + docs closed  

## Out of scope

- Auth, alerts, trading  
- Frontend web pumps UI  
- Backend OpenAPI schema tighten (optional follow-up)  
- AI narrative of pumps  

## MR grouping

`(A+1–2)` · `(3–4)` · `(5–6)` · `(7)`
