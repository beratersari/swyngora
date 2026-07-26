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
// feature or page
import { useListSpotMarketsQuery } from '@/libs/api';
```

## Codegen

```bash
npm run codegen:api   # writes into libs/api/generated/
```

After OpenAPI changes in `backend/api/openapi/openapi.yaml`, regenerate in the same MR.
