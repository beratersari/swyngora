import { useCallback, useMemo, useState } from 'react';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import { useListSpotMarketsQuery, type MarketExchange } from '@/libs/api';
import {
  loadDeskTapeSource,
  saveDeskTapeSource,
  DESK_TAPE_VENUE_LIMIT,
  type DeskTapeSource,
} from '@/components/organisms/DeskPriceTape';
import { toTickerTapeItem, type TickerTapeItem } from '@/components/molecules/TickerTape';
import { defaultQuoteForExchange } from '@/libs/utils';
import { useDisplayCurrency } from './displayCurrency';
import { useDocumentVisible } from './useDocumentVisible';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';

export function useDeskPriceTape(): {
  source: DeskTapeSource;
  setSource: (next: DeskTapeSource) => void;
  items: TickerTapeItem[];
  isLoading: boolean;
  paused: boolean;
} {
  const visible = useDocumentVisible();
  const livePrices = useRealtimeConnected();
  const { currency, rates } = useDisplayCurrency();
  const [source, setSourceState] = useState<DeskTapeSource>(loadDeskTapeSource);
  const setSource = useCallback((next: DeskTapeSource) => {
    setSourceState(next);
    saveDeskTapeSource(next);
  }, []);

  const venue = source === 'watchlist' ? undefined : source;

  const venueQuery = useListSpotMarketsQuery(
    {
      exchange: (venue ?? 'binance') as MarketExchange,
      quote: defaultQuoteForExchange(venue ?? 'binance'),
      sort: 'quoteVolume',
      order: 'desc',
      limit: DESK_TAPE_VENUE_LIMIT,
      offset: 0,
      status: 'TRADING',
    },
    {
      skip: !venue,
      pollingInterval: !visible ? 0 : livePrices ? SPOT_LIST_WS_REST_POLL_MS : DEFAULT_SPOT_POLL_MS,
      refetchOnFocus: true,
    },
  );

  const venueSymbols = useMemo(() => {
    if (!venue) return [];
    return (venueQuery.data?.items ?? [])
      .map((row) => ({
        exchange: venue,
        symbol: (row.symbol ?? '').toUpperCase(),
      }))
      .filter((it) => it.symbol);
  }, [venue, venueQuery.data?.items]);
  usePriceSubscription(venueSymbols, visible && Boolean(venue));

  const display = useMemo(() => ({ currency, rates }), [currency, rates]);

  const items = useMemo(() => {
    if (!venue) return [];
    const out: TickerTapeItem[] = [];
    for (const row of venueQuery.data?.items ?? []) {
      const item = toTickerTapeItem(
        {
          exchange: venue,
          symbol: row.symbol,
          lastPrice: row.lastPrice,
          priceChangePercent: row.priceChangePercent,
        },
        display,
      );
      if (item) out.push(item);
    }
    return out;
  }, [venue, venueQuery.data?.items, display]);

  const isLoading = Boolean(venue && (venueQuery.isLoading || (venueQuery.isFetching && items.length === 0)));

  return { source, setSource, items, isLoading, paused: !visible };
}
