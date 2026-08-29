import { useCallback, useMemo, useState } from 'react';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import { useListSpotMarketsQuery, type MarketExchange, type SpotMarket } from '@/libs/api';
import {
  loadDeskTapeSource,
  saveDeskTapeSource,
  DESK_TAPE_VENUE_LIMIT,
  type DeskTapeSource,
} from '@/components/organisms/DeskPriceTape';
import { toTickerTapeItem, type TickerTapeItem } from '@/components/molecules/TickerTape';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import type { DisplayCurrency, FxRatesMap } from '@/libs/utils';
import { useDisplayCurrency } from './displayCurrency';
import { useDocumentVisible } from './useDocumentVisible';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';

/** Build tape rows only from the list that belongs to `venue`. */
export function deskTapeItemsFromList(
  venue: string | undefined,
  list: readonly SpotMarket[] | undefined,
  display: { currency: DisplayCurrency; rates?: FxRatesMap },
  sourceExchange?: string,
): TickerTapeItem[] {
  if (!venue) return [];
  if (sourceExchange && sourceExchange !== venue) return [];
  const out: TickerTapeItem[] = [];
  for (const row of list ?? []) {
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
}

export function useDeskPriceTape(): {
  source: DeskTapeSource;
  setSource: (next: DeskTapeSource) => void;
  items: TickerTapeItem[];
  isLoading: boolean;
  paused: boolean;
} {
  const visible = useDocumentVisible();
  const livePrices = useRealtimeConnected();
  const { currency, rates, nativeQuote } = useDisplayCurrency();
  const [source, setSourceState] = useState<DeskTapeSource>(loadDeskTapeSource);
  const setSource = useCallback((next: DeskTapeSource) => {
    setSourceState(next);
    saveDeskTapeSource(next);
  }, []);

  const venue = source === 'watchlist' ? undefined : source;

  const venueQuery = useListSpotMarketsQuery(
    {
      exchange: (venue ?? 'binance') as MarketExchange,
      quote: nativeQuote(venue ?? 'binance'),
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

  const liveList = rtkCurrent(venueQuery);
  const sourceExchange = venueQuery.originalArgs?.exchange;
  const venueSymbols = useMemo(() => {
    if (!venue || (sourceExchange && sourceExchange !== venue)) return [];
    return (liveList?.items ?? [])
      .map((row) => ({
        exchange: venue,
        symbol: (row.symbol ?? '').toUpperCase(),
      }))
      .filter((it) => it.symbol);
  }, [venue, liveList?.items, sourceExchange]);
  usePriceSubscription(venueSymbols, visible && Boolean(venue));

  const display = useMemo(() => ({ currency, rates }), [currency, rates]);

  const items = useMemo(
    () => deskTapeItemsFromList(venue, liveList?.items, display, sourceExchange),
    [venue, liveList?.items, display, sourceExchange],
  );

  const isLoading = Boolean(venue && rtkCurrentPending(venueQuery));

  return { source, setSource, items, isLoading, paused: !visible };
}
