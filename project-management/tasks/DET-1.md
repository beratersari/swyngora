# DET-1: RTK detail endpoints (implementation)

| Field | Value |
|---|---|
| **ID** | DET-1 |
| **Epic** | coin-detail-and-indicators |
| **Status** | done |
| **Area** | frontend |
| **Type** | implementation |
| **Blocked by** | DET-A, DET-B accepted |

## Summary

Add RTK Query endpoints under `libs/api/endpoints/marketApi.ts`: intervals, candles, ticker/24h, supply, indicators. OpenAPI-derived types. No page UI in this task.

## Status

**done.** Implemented in `frontend/src/libs/api/endpoints/marketApi.ts` (+ exports in `libs/api/index.ts`, tags in `baseApi.ts`).
