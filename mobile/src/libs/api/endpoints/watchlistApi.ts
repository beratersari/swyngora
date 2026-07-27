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

export type ReplaceWatchlistArg = {
  items: { exchange?: string; symbol: string; note?: string }[];
};

function withClientIdQuery(extra?: Record<string, string>): Record<string, string> {
  const clientId = getOrCreateClientId();
  return { clientId, ...(extra ?? {}) };
}

export const watchlistApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getWatchlist: build.query<Watchlist, void>({
      query: () => ({
        url: '/api/v1/watchlist',
        params: withClientIdQuery(),
      }),
      providesTags: (result) => [
        { type: 'Watchlist' as const, id: result?.clientId ?? 'LIST' },
        { type: 'Watchlist' as const, id: 'LIST' },
      ],
    }),

    addWatchlistItem: build.mutation<Watchlist, AddWatchlistItemArg>({
      query: (arg) => ({
        url: '/api/v1/watchlist/items',
        method: 'POST',
        body: {
          clientId: getOrCreateClientId(),
          exchange: arg.exchange ?? 'binance',
          symbol: arg.symbol,
          ...(arg.note ? { note: arg.note } : {}),
        },
      }),
      invalidatesTags: [{ type: 'Watchlist', id: 'LIST' }],
    }),

    removeWatchlistItem: build.mutation<Watchlist, RemoveWatchlistItemArg>({
      query: (arg) => ({
        url: '/api/v1/watchlist/items',
        method: 'DELETE',
        params: withClientIdQuery({
          exchange: arg.exchange,
          symbol: arg.symbol,
        }),
      }),
      invalidatesTags: [{ type: 'Watchlist', id: 'LIST' }],
    }),

    replaceWatchlist: build.mutation<Watchlist, ReplaceWatchlistArg>({
      query: (arg) => ({
        url: '/api/v1/watchlist',
        method: 'PUT',
        body: {
          clientId: getOrCreateClientId(),
          items: arg.items,
        },
      }),
      invalidatesTags: [{ type: 'Watchlist', id: 'LIST' }],
    }),
  }),
});

export const {
  useGetWatchlistQuery,
  useLazyGetWatchlistQuery,
  useAddWatchlistItemMutation,
  useRemoveWatchlistItemMutation,
  useReplaceWatchlistMutation,
} = watchlistApi;
