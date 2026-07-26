# libs/api

API layer: RTK Query + OpenAPI.

```text
libs/api/
├── baseApi.ts           # createApi + fetchBaseQuery + tagTypes
├── store.ts             # configureStore (api reducer + middleware)
├── hooks.ts             # useAppDispatch / useAppSelector (typed)
├── endpoints/           # injectEndpoints per domain (marketApi.ts, …)
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
```

Endpoints live in `endpoints/marketApi.ts` and `endpoints/healthApi.ts`.

## Codegen

```bash
npm run codegen:api   # writes into libs/api/generated/
```

After OpenAPI changes in `backend/api/openapi/openapi.yaml`, regenerate in the same MR.
