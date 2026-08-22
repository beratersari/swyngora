import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { env } from '@/config/env';
import { getOrCreateClientId } from '@/libs/utils/clientId';

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({
    baseUrl: env.apiBaseUrl,
    prepareHeaders: (headers) => {
      headers.set('X-Client-Id', getOrCreateClientId());
      return headers;
    },
  }),
  tagTypes: [
    'Alert',
    'SpotList',
    'Exchange',
    'ProductTag',
    'Watchlist',
    'Health',
    'Candle',
    'Ticker',
    'OrderBook',
    'OrderHeatmap',
    'Supply',
    'Holders',
    'Interval',
    'Indicator',
    'Pump',
    'Portfolio',
    'AccountAPIKey',
    'ExportJob',
    'RecurringBuy',
    'WatchlistShare',
    'ScannerRule',
    'ScannerResult',
    'ScannerBacktest',
    'SwingSetup',
  ],
  endpoints: () => ({}),
});
