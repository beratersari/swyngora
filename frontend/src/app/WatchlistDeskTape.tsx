import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { DEFAULT_DETAIL_TICKER_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import { TickerTape, toTickerTapeItem, type TickerTapeItem } from '@/components/molecules/TickerTape';
import { useGetTicker24hQuery, useGetWatchlistQuery, type MarketExchange } from '@/libs/api';
import { useDisplayCurrency, useDocumentVisible } from '@/libs/hooks';
import { usePriceSubscription } from '@/libs/realtime';

const WATCHLIST_TAPE_MAX = 50;

function rowKey(exchange: string, symbol: string): string {
  return `${exchange}:${symbol}`;
}

/** Per-symbol ticker fetch; lifts quotes so the shared marquee can scroll. */
export function WatchlistDeskTape({
  ariaLabel,
  emptyLabel,
  paused = false,
}: {
  ariaLabel: string;
  emptyLabel: string;
  paused?: boolean;
}) {
  const { t } = useTranslation('common');
  const visible = useDocumentVisible();
  const [quotes, setQuotes] = useState<Record<string, TickerTapeItem>>({});
  const wl = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const rows = useMemo(
    () =>
      (wl.data?.items ?? [])
        .map((it) => ({
          exchange: (it.exchange ?? '').toLowerCase(),
          symbol: (it.symbol ?? '').toUpperCase(),
        }))
        .filter((it) => it.exchange && it.symbol)
        .slice(0, WATCHLIST_TAPE_MAX),
    [wl.data?.items],
  );

  const onItem = useCallback((exchange: string, symbol: string, item: TickerTapeItem) => {
    const key = rowKey(exchange, symbol);
    setQuotes((prev) => {
      const cur = prev[key];
      if (
        cur &&
        cur.lastPrice === item.lastPrice &&
        cur.changePercent === item.changePercent &&
        cur.href === item.href
      ) {
        return prev;
      }
      return { ...prev, [key]: item };
    });
  }, []);

  const items = useMemo(
    () => rows.map((row) => quotes[rowKey(row.exchange, row.symbol)]).filter((item): item is TickerTapeItem => Boolean(item)),
    [rows, quotes],
  );

  if (wl.isLoading && rows.length === 0) {
    return <span style={{ padding: '0 16px', fontSize: 12 }}>{t('status.loading')}</span>;
  }
  if (rows.length === 0) {
    return (
      <span data-testid="watchlist-tape-empty" style={{ padding: '0 16px', fontSize: 12 }}>
        {emptyLabel}
      </span>
    );
  }

  return (
    <>
      {rows.map((row) => (
        <WatchlistDeskTapeProbe
          key={rowKey(row.exchange, row.symbol)}
          exchange={row.exchange}
          symbol={row.symbol}
          visible={visible}
          onItem={onItem}
        />
      ))}
      {items.length > 0 ? (
        <TickerTape items={items} ariaLabel={ariaLabel} paused={paused} />
      ) : (
        <span style={{ padding: '0 16px', fontSize: 12 }}>{t('status.loading')}</span>
      )}
    </>
  );
}

function WatchlistDeskTapeProbe({
  exchange,
  symbol,
  visible,
  onItem,
}: {
  exchange: string;
  symbol: string;
  visible: boolean;
  onItem: (exchange: string, symbol: string, item: TickerTapeItem) => void;
}) {
  const { currency, rates } = useDisplayCurrency();
  const { connected } = usePriceSubscription([{ exchange, symbol }], visible);
  const ticker = useGetTicker24hQuery(
    { exchange: exchange as MarketExchange, symbol },
    {
      skip: !exchange || !symbol,
      pollingInterval: !visible ? 0 : connected ? SPOT_LIST_WS_REST_POLL_MS : DEFAULT_DETAIL_TICKER_POLL_MS,
      refetchOnFocus: true,
    },
  );
  const item = toTickerTapeItem(
    {
      exchange,
      symbol,
      lastPrice: ticker.data?.lastPrice,
      priceChangePercent: ticker.data?.priceChangePercent,
    },
    { currency, rates },
  );

  useEffect(() => {
    if (item) onItem(exchange, symbol, item);
  }, [exchange, symbol, item, onItem]);

  return null;
}
