import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { CompressOutlined, ExpandOutlined } from '@ant-design/icons';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PageHeader } from '@/components/molecules/PageHeader';
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
import { defaultQuoteForExchange, parseExchangeParamOrDefault, rtkCurrent } from '@/libs/utils';
import { BoardWrap, Field, PageStack, Toolbar, ToolbarLeft, ToolbarRight } from './HeatmapPage.styles';

const VENUES: MarketExchange[] = ['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'];

export function HeatmapPage() {
  const { t } = useTranslation(['heatmap', 'common']);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const exchange = parseExchangeParamOrDefault(searchParams.get('exchange') ?? undefined);
  const quote = searchParams.get('quote') || defaultQuoteForExchange(exchange);
  const [metric, setMetric] = useState<HeatmapMetric>('marketCap');
  const [fullscreen, setFullscreen] = useState(false);
  const boardRef = useRef<HTMLDivElement>(null);

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
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next && next !== 'binance') p.set('exchange', next);
        else p.delete('exchange');
        p.delete('quote');
        return p;
      },
      { replace: true },
    );
  };

  const setQuote = (next: string) => {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        const def = defaultQuoteForExchange(exchange);
        if (next && next !== def) p.set('quote', next);
        else p.delete('quote');
        return p;
      },
      { replace: true },
    );
  };

  useEffect(() => {
    const onChange = () => setFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', onChange);
    return () => document.removeEventListener('fullscreenchange', onChange);
  }, []);

  const toggleFullscreen = () => {
    const el = boardRef.current;
    if (!el) return;
    if (document.fullscreenElement) {
      void document.exitFullscreen();
      return;
    }
    void el.requestFullscreen?.();
  };

  return (
    <PageStack>
      <PageHeader title={t('heatmap:title')} subtitle={t('heatmap:subtitle')} />

      <Toolbar>
        <ToolbarLeft>
          <ExchangeTabs exchanges={VENUES} value={exchange} onChange={setVenue} />
          <Field>
            <span id="heatmap-quote-label">{t('heatmap:quote')}</span>
            <Select
              aria-labelledby="heatmap-quote-label"
              value={quote}
              size="middle"
              style={{ minWidth: 96 }}
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
        </ToolbarLeft>
        <ToolbarRight>
          <Field>
            <span id="heatmap-metric-label">{t('heatmap:sizeBy')}</span>
            <Select
              aria-labelledby="heatmap-metric-label"
              value={metric}
              size="middle"
              style={{ minWidth: 148 }}
              options={[
                { value: 'marketCap', label: t('heatmap:metric.marketCap') },
                { value: 'quoteVolume', label: t('heatmap:metric.volume') },
              ]}
              onChange={(v) => setMetric(v as HeatmapMetric)}
            />
          </Field>
          <Button
            type="default"
            icon={fullscreen ? <CompressOutlined /> : <ExpandOutlined />}
            onClick={toggleFullscreen}
          >
            {fullscreen ? t('heatmap:exitFullscreen') : t('heatmap:fullscreen')}
          </Button>
        </ToolbarRight>
      </Toolbar>

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

      <BoardWrap ref={boardRef}>
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
