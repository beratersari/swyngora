import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { env } from '@/config/env';

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({
    baseUrl: env.apiBaseUrl,
    prepareHeaders: (headers) => {
      // Watchlist OpenAPI uses X-Client-Id; optional for local/dev.
      if (env.clientId) {
        headers.set('X-Client-Id', env.clientId);
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
    'Candle',
    'Ticker',
    'Supply',
    'Interval',
    'Indicator',
    'Pump',
  ],
  endpoints: () => ({}),
});
