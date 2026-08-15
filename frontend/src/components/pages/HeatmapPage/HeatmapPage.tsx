import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { ExchangeTabs } from '@/components/organisms/ExchangeTabs';
import {
  PriceChangeHeatmap,
  type HeatmapMetric,
} from '@/components/organisms/PriceChangeHeatmap';
import { HEATMAP_LEGEND_GRADIENT, HEATMAP_MAX_TILES } from '@/components/organisms/PriceChangeHeatmap/PriceChangeHeatmap.constants';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import {
  rtkErrorMessage,
  useListSpotMarketsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';
import { defaultQuoteForExchange, rtkCurrent } from '@/libs/utils';
import {
  Chrome,
  ChromeLeft,
  Field,
  Legend,
  LegendBar,
  LegendTick,
  PageStack,
} from './HeatmapPage.styles';

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
      <Text variant="h4" color="primary" as="h1">
        {t('heatmap:title')}
      </Text>

      <Chrome>
        <ChromeLeft>
          <ExchangeTabs exchanges={VENUES} value={exchange} onChange={setVenue} />
          <Field>
            <Text variant="caption" color="secondary" id="heatmap-quote-label">
              {t('heatmap:quote')}
            </Text>
            <Select
              aria-labelledby="heatmap-quote-label"
              value={quote}
              style={{ minWidth: 100 }}
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
            <Text variant="caption" color="secondary" id="heatmap-metric-label">
              {t('heatmap:sizeBy')}
            </Text>
            <Select
              aria-labelledby="heatmap-metric-label"
              value={metric}
              style={{ minWidth: 150 }}
              options={[
                { value: 'quoteVolume', label: t('heatmap:metric.volume') },
                { value: 'marketCap', label: t('heatmap:metric.marketCap') },
              ]}
              onChange={(v) => setMetric(v as HeatmapMetric)}
            />
          </Field>
        </ChromeLeft>
        <Legend aria-hidden>
          <LegendTick>−8%</LegendTick>
          <LegendBar style={{ background: `linear-gradient(90deg, ${HEATMAP_LEGEND_GRADIENT})` }} />
          <LegendTick>+8%</LegendTick>
        </Legend>
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

      <PriceChangeHeatmap
        items={items}
        metric={metric}
        isLoading={listQuery.isLoading || listQuery.isFetching}
        onOpen={(ex, sym) => navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(sym)}`)}
      />
    </PageStack>
  );
}
