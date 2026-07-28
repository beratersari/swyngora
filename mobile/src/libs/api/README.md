# libs/api

RTK Query + OpenAPI types for the mobile client.

- `baseApi.ts` — createApi (`X-Client-Id` header for watchlist)
- `endpoints/marketApi.ts` — exchanges, spot, candles, ticker, indicators
- `endpoints/watchlistApi.ts` — get/add/remove/replace watchlist
- `endpoints/healthApi.ts` — liveness
- `endpoints/aiApi.ts` — `POST /api/v1/ai/chat` assistant mutation
- `generated/` — openapi-typescript output (read-only)
