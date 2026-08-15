import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ExchangeTabs } from '@/components/organisms/ExchangeTabs';
import {
  PriceChangeHeatmap,
  type HeatmapMetric,
} from '@/components/organisms/PriceChangeHeatmap';
import { HEATMAP_MAX_TILES } from '@/components/organisms/PriceChangeHeatmap/PriceChangeHeatmap.constants';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import {
  rtkErrorMessage,
  useListSpotMarketsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';
import { defaultQuoteForExchange, rtkCurrent } from '@/libs/utils';
import { BoardWrap, Chrome, ChromeLeft, Field, PageStack, Title } from './HeatmapPage.styles';

const VENUES: MarketExchange[] = ['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'];

export function HeatmapPage() {
  const { t } = useTranslation(['heatmap', 'common']);
  const navigate = useNavigate();
  const visible = useDocumentVisible();
  const [exchange, setExchange] = useState<MarketExchange>('binance');
  const [quote, setQuote] = useState(defaultQuoteForExchange('binance'));
  const [metric, setMetric] = useState<HeatmapMetric>('quoteVolume');

  const wsLive = useRealtimeConnected();
  const poll = !visible ? 0 : wsLive ? SPOT_LIST_WS_REST_POLL_MS : DEFAULT_SPOT_POLL_MS;

  const listQuery = useListSpotMarketsQuery(
    {
      exchange,
      quote,
      sort: metric === 'marketCap' ? 'marketCapCirculating' : 'quoteVolume',
      order: 'desc',
      limit: HEATMAP_MAX_TILES,
      offset: 0,
      status: 'TRADING',
    },
    { pollingInterval: poll, refetchOnFocus: true },
  );

  const liveList = rtkCurrent(listQuery);
  const symbols = useMemo(
    () => (liveList?.items ?? []).map((row) => row.symbol).filter((s): s is string => Boolean(s)),
    [liveList?.items],
  );
  usePriceSubscription(
    symbols.map((symbol) => ({ exchange, symbol })),
    visible && symbols.length > 0,
  );

  const items = useMemo(
    () =>
      (liveList?.items ?? []).map((row) => ({
        symbol: row.symbol ?? '',
        exchange,
        lastPrice: row.lastPrice,
        priceChangePercent: row.priceChangePercent,
        quoteVolume: row.quoteVolume,
        marketCapCirculating: row.marketCapCirculating,
      })),
    [exchange, liveList?.items],
  );

  const setVenue = (next: MarketExchange) => {
    setExchange(next);
    setQuote(defaultQuoteForExchange(next));
  };

  return (
    <PageStack>
      <Chrome>
        <ChromeLeft>
          <Title>{t('heatmap:title')}</Title>
          <ExchangeTabs exchanges={VENUES} value={exchange} onChange={setVenue} />
          <Field>
            <span id="heatmap-quote-label">{t('heatmap:quote')}</span>
            <Select
              aria-labelledby="heatmap-quote-label"
              value={quote}
              size="small"
              style={{ minWidth: 88 }}
              options={
                exchange === 'nasdaq'
                  ? [{ value: 'USD', label: 'USD' }]
                  : exchange === 'bist'
                    ? [{ value: 'TRY', label: 'TRY' }]
                    : exchange === 'coinbase'
                      ? [
                          { value: 'USD', label: 'USD' },
                          { value: 'USDT', label: 'USDT' },
                        ]
                      : [
                          { value: 'USDT', label: 'USDT' },
                          { value: 'USDC', label: 'USDC' },
                        ]
              }
              onChange={setQuote}
            />
          </Field>
          <Field>
            <span id="heatmap-metric-label">{t('heatmap:sizeBy')}</span>
            <Select
              aria-labelledby="heatmap-metric-label"
              value={metric}
              size="small"
              style={{ minWidth: 132 }}
              options={[
                { value: 'quoteVolume', label: t('heatmap:metric.volume') },
                { value: 'marketCap', label: t('heatmap:metric.marketCap') },
              ]}
              onChange={(v) => setMetric(v as HeatmapMetric)}
            />
          </Field>
        </ChromeLeft>
      </Chrome>

      {listQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('heatmap:loadFailed')}
          description={rtkErrorMessage(listQuery.error, { resource: t('heatmap:resource') })}
          action={
            <Button size="small" onClick={() => void listQuery.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      <BoardWrap>
        <PriceChangeHeatmap
          items={items}
          metric={metric}
          isLoading={listQuery.isLoading || listQuery.isFetching}
          onOpen={(ex, sym) => navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(sym)}`)}
        />
      </BoardWrap>
    </PageStack>
  );
}
