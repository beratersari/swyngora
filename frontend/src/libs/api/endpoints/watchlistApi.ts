import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';
import { getOrCreateClientId } from '@/libs/utils/clientId';

export type Watchlist = components['schemas']['Watchlist'];
export type WatchlistItem = NonNullable<Watchlist['items']>[number];

export type AddWatchlistItemArg = {
  exchange?: string;
  symbol: string;
  note?: string;
};

export type RemoveWatchlistItemArg = {
  exchange: string;
  symbol: string;
};

export const watchlistApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getWatchlist: build.query<Watchlist, void>({
      query: () => ({
        url: '/api/v1/watchlist',
        params: { clientId: getOrCreateClientId() },
      }),
      providesTags: ['Watchlist'],
    }),
    addWatchlistItem: build.mutation<Watchlist, AddWatchlistItemArg>({
      query: (body) => ({
        url: '/api/v1/watchlist/items',
        method: 'POST',
        body: {
          clientId: getOrCreateClientId(),
          exchange: body.exchange ?? 'binance',
          symbol: body.symbol,
          note: body.note,
        },
      }),
      invalidatesTags: ['Watchlist'],
    }),
    removeWatchlistItem: build.mutation<Watchlist, RemoveWatchlistItemArg>({
      query: ({ exchange, symbol }) => ({
        url: '/api/v1/watchlist/items',
        method: 'DELETE',
        params: {
          exchange,
          symbol,
          clientId: getOrCreateClientId(),
        },
      }),
      invalidatesTags: ['Watchlist'],
    }),
  }),
});

export const {
  useGetWatchlistQuery,
  useAddWatchlistItemMutation,
  useRemoveWatchlistItemMutation,
} = watchlistApi;
