# libs/api

API layer: RTK Query + OpenAPI.

```text
libs/api/
├── baseApi.ts           # createApi + fetchBaseQuery + tagTypes + X-Client-Id header
├── store.ts             # configureStore (api reducer + middleware)
├── hooks.ts             # useAppDispatch / useAppSelector (typed)
├── endpoints/           # injectEndpoints per domain
│   ├── marketApi.ts     # RTK endpoints
│   ├── marketApi.helpers.ts  # pure transforms + cache tag ids (unit-tested)
│   ├── marketApi.types.ts
│   ├── portfolioApi.ts  # paper portfolio + performance history
│   ├── aiApi.ts         # POST /api/v1/ai/chat (non-stream fallback)
│   └── healthApi.ts
├── aiChatStream.ts      # POST /api/v1/ai/chat/stream NDJSON (not RTK)
├── generated/           # OpenAPI codegen output — DO NOT hand-edit
└── index.ts             # public barrel
```

## Usage

```ts
// markets list
import { useListSpotMarketsQuery } from '@/libs/api';

// coin detail
import {
  useGetCandlesQuery,
  useGetTicker24hQuery,
  useGetSupplyQuery,
  useListIntervalsQuery,
  useGetIndicatorsQuery,
} from '@/libs/api';

// pumps / watchlist / AI chat / scanner
import {
  useGetPumpEventsQuery,
  useScanPumpEventsQuery,
  useGetWatchlistQuery,
  useAddWatchlistItemMutation,
  usePostAiChatMutation,
  streamAiChat,
  useGetPortfolioPerformanceQuery,
  useListScannerRulesQuery,
  useListScannerResultsQuery,
} from '@/libs/api';
```

### Cache tags

- **SpotList** tags are **arg-scoped** (`spotListTagId`) plus a shared `LIST` id so invalidation can target one filter set or all lists.
- Detail series (candles / ticker / indicators / supply) use composite ids (exchange/symbol/…).
- **Pump** tag type covers `getPumpEvents` / `scanPumpEvents`.
- **ScannerRule / ScannerResult / ScannerBacktest** cover `/api/v1/scanner/*`.
- **Portfolio** covers paper snapshot + performance series.
- **Watchlist** tag is registered. `prepareHeaders` always sets `X-Client-Id` via `getOrCreateClientId()` (optional `VITE_CLIENT_ID` overrides the generated id).
- Live prices/portfolio patch these caches from `libs/realtime` (WebSocket). Polling is only the fallback while disconnected.

## Auth (dev proxy)

The browser never embeds `API_AUTH_TOKEN`. When the backend enables `API_AUTH_TOKEN`, local Vite can inject the secret **only on the proxy hop**:

```bash
# shell that runs `npm run dev` (not VITE_* → not in the client bundle)
export API_AUTH_TOKEN=dev-secret
# or: export VITE_DEV_API_AUTH_TOKEN=dev-secret
npm run dev
```

Without that, market routes stay public; watchlist/AI return 401 when the token is set.

## Codegen

```bash
npm run codegen:api   # writes into libs/api/generated/
```

After OpenAPI changes in `backend/api/openapi/openapi.yaml`, regenerate in the same MR. Pump event/hit shapes are named schemas (`PumpEvent`, `PumpScanHit`, …).
