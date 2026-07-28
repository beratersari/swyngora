# Epic H: Mobile batch indicators (list RSI/EMA)

**Priority:** P1 (mobile track — after pumps)  
**Status:** done  
**Depends on:** Markets + coin detail + favorites (done)  
**Branch:** `feature/mobile-batch-indicators` (from latest `develop` / after pumps merges)  
**Design plan:** `docs/design/mobile-batch-indicators.md`  
**Feature:** `docs/features/mobile-batch-indicators.md`  
**Analysis:** `project-management/tasks/mobile/batch-indicators/MBIND-A.md`  
**Tasks folder:** `project-management/tasks/mobile/batch-indicators/`

## Goal

Enrich mobile **list screens** (Favorites first, Markets second) with **latest RSI** (and optional EMA) using existing  
`POST /api/v1/market/indicators/batch` — avoid N+1 `GET /indicators` calls. Keep coin-detail series on single-symbol API.

Atomic Design (kebab-case) + View + ViewModel; primary runtime **Chrome**.

## APIs (existing backend)

| UI | Path |
|----|------|
| Batch latest RSI/EMA | `POST /api/v1/market/indicators/batch` |
| Detail series (unchanged) | `GET /api/v1/market/indicators` |
| Intervals / exchanges | existing market endpoints |

**No backend feature work** for mobile v1 (OpenAPI already describes batch).

## Tasks

- [x] MBIND-A — API field matrix analysis  
- [x] MBIND-1 — RTK postIndicatorsBatch  
- [x] MBIND-2 — chunk/group/format helpers + tests  
- [x] MBIND-3 — RSI badge + row prop extensions  
- [x] MBIND-4 — Favorites batch enrichment (P0)  
- [x] MBIND-5 — Markets list batch enrichment (P1)  
- [x] MBIND-6 — Loading / partial failure / disclaimer  
- [x] MBIND-7 — Tests polish + docs/board/changelog  

## Acceptance

- Favorites show RSI from batch for saved pairs (≤50 per exchange request)  
- Multi-exchange favorites → one batch per exchange  
- Partial item failures show `—` without breaking prices  
- AppState / focus pause batch polling  
- Markets first page (or controlled set) can show RSI  
- Coin detail still uses single-symbol indicators  
- Tests + docs closed  

## Out of scope

- Auth, alerts on RSI thresholds, trading  
- Replacing detail chart series with batch  
- User-editable RSI/EMA periods on lists (fixed 14 / 12,26)  
- Frontend web markets RSI columns (separate backlog)  
- MCP `get_indicators_batch` (optional follow-up; single `get_indicators` exists)  

## MR grouping

`(A+1–2)` · `(3–4)` · `(5–6)` · `(7)`
