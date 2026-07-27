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
│   └── healthApi.ts
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

// pumps (OpenAPI-backed; optional UI later)
import { useGetPumpEventsQuery, useScanPumpEventsQuery } from '@/libs/api';
```

### Cache tags

- **SpotList** tags are **arg-scoped** (`spotListTagId`) plus a shared `LIST` id so invalidation can target one filter set or all lists.
- Detail series (candles / ticker / indicators / supply) use composite ids (exchange/symbol/…).
- **Pump** tag type covers `getPumpEvents` / `scanPumpEvents`.
- **Watchlist** tag is registered; send `VITE_CLIENT_ID` so `prepareHeaders` sets `X-Client-Id` when wiring mutations.

## Codegen

```bash
npm run codegen:api   # writes into libs/api/generated/
```

After OpenAPI changes in `backend/api/openapi/openapi.yaml`, regenerate in the same MR.
