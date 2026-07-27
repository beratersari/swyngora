import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { env } from '@/config/env';
import { peekClientId, getOrCreateClientId } from '@/libs/utils/clientId';

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({
    baseUrl: env.apiBaseUrl,
    prepareHeaders: (headers) => {
      // Prefer existing id; create only if a request is already going out after hydrate
      const id = peekClientId() ?? getOrCreateClientId();
      if (id) {
        headers.set('X-Client-Id', id);
      }
      return headers;
    },
  }),
  tagTypes: [
    'SpotList',
    'Exchange',
    'ProductTag',
    'Watchlist',
    'Health',
    'Interval',
    'Candle',
    'Ticker',
    'Supply',
    'Indicator',
    'Pump',
  ],
  endpoints: () => ({}),
});
