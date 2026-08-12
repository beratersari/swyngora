import {
  useListSpotMarketsQuery,
  type MarketExchange,
  type SpotMarket,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks/useDocumentVisible';
import { usePriceSubscription } from '@/libs/realtime';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';

/** Prefer exact symbol match from a spot search result list. */
export function pickSpotForSymbol(
  items: SpotMarket[] | undefined,
  symbol: string,
): SpotMarket | undefined {
  if (!items?.length || !symbol) return undefined;
  const target = symbol.trim().toUpperCase();
  const targetNorm = target.replace(/[-_/]/g, '');

  const exact =
    items.find((it) => (it.symbol ?? '').toUpperCase() === target) ??
    items.find(
      (it) => (it.symbol ?? '').toUpperCase().replace(/[-_/]/g, '') === targetNorm,
    );
  if (exact) return exact;

  return items.find((it) => {
    const s = (it.symbol ?? '').toUpperCase();
    return s === target || s.replace(/[-_/]/g, '') === targetNorm;
  });
}

/**
 * Live spot snapshot for one watchlist pair.
 * Used by page-level cell components only (not organisms).
 * Polls only while the document is visible (§6.6).
 */
export function useWatchlistSpot(
  exchange: string,
  symbol: string,
): {
  spot: SpotMarket | undefined;
  isLoading: boolean;
  isError: boolean;
} {
  const visible = useDocumentVisible();
  const { connected: livePrices } = usePriceSubscription(
    exchange && symbol ? [{ exchange, symbol }] : [],
    visible,
  );
  const q = useListSpotMarketsQuery(
    {
      exchange: exchange as MarketExchange,
      q: symbol,
      limit: 25,
      status: 'TRADING',
      sort: 'quoteVolume',
      order: 'desc',
    },
    {
      skip: !exchange || !symbol,
      pollingInterval: !visible ? 0 : livePrices ? SPOT_LIST_WS_REST_POLL_MS : DEFAULT_SPOT_POLL_MS,
      refetchOnFocus: true,
    },
  );
  const spot = pickSpotForSymbol(q.data?.items, symbol);
  const isLoading = Boolean(q.isLoading || (q.isFetching && !spot));
  return { spot, isLoading, isError: Boolean(q.isError) };
}
