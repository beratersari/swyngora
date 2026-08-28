import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Segmented, Select } from 'antd';
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
import { RSIHeatmap } from '@/components/organisms/RSIHeatmap';
import { RSI_HEAT_INTERVALS, RSI_HEAT_TOPS } from '@/components/organisms/RSIHeatmap/constants';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import {
  rtkErrorMessage,
  useGetRSIHeatmapQuery,
  useListSpotMarketsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';
import { defaultQuoteForExchange, parseExchangeParamOrDefault, rtkCurrent } from '@/libs/utils';
import { BoardWrap, Field, PageStack, Toolbar, ToolbarLeft, ToolbarRight } from './HeatmapPage.styles';

const RSI_HEAT_POLL_MS = 60_000;

const VENUES: MarketExchange[] = ['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'];

export function HeatmapPage() {
  const { t } = useTranslation(['heatmap', 'common']);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const exchange = parseExchangeParamOrDefault(searchParams.get('exchange') ?? undefined);
  const quote = searchParams.get('quote') || defaultQuoteForExchange(exchange);
  const view = searchParams.get('view') === 'rsi' ? 'rsi' : 'price';
  const rawInterval = searchParams.get('interval') ?? '';
  const rsiInterval = (RSI_HEAT_INTERVALS as readonly string[]).includes(rawInterval)
    ? rawInterval
    : '1h';
  const rsiTop = Number(searchParams.get('top')) === 50 ? 50 : 100;
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
    { skip: view !== 'price', pollingInterval: view === 'price' ? poll : 0, refetchOnFocus: true },
  );

  const liveList = rtkCurrent(listQuery);
  const symbols = useMemo(
    () => (liveList?.items ?? []).map((row) => row.symbol).filter((s): s is string => Boolean(s)),
    [liveList?.items],
  );
  usePriceSubscription(
    symbols.map((symbol) => ({ exchange, symbol })),
    view === 'price' && visible && symbols.length > 0,
  );

  const rsiQuery = useGetRSIHeatmapQuery(
    {
      exchange,
      quote,
      interval: rsiInterval,
      limit: rsiTop,
      sort: 'marketCapCirculating',
    },
    { skip: view !== 'rsi' || !visible, pollingInterval: view === 'rsi' && visible ? RSI_HEAT_POLL_MS : 0, refetchOnFocus: true },
  );
  const rsiData = rtkCurrent(rsiQuery);

  const setView = (next: string) => {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next === 'rsi') p.set('view', 'rsi');
        else p.delete('view');
        return p;
      },
      { replace: true },
    );
  };

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

  const setRsiInterval = (next: string) => {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next && next !== '1h') p.set('interval', next);
        else p.delete('interval');
        return p;
      },
      { replace: true },
    );
  };

  const setRsiTop = (next: number) => {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next === 50) p.set('top', '50');
        else p.delete('top');
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
      <PageHeader
        title={view === 'rsi' ? t('heatmap:rsi.title') : t('heatmap:title')}
        subtitle={view === 'rsi' ? t('heatmap:rsi.subtitle') : t('heatmap:subtitle')}
      />

      <Toolbar>
        <ToolbarLeft>
          <Segmented
            value={view}
            onChange={(v) => setView(String(v))}
            options={[
              { value: 'price', label: t('heatmap:view.price') },
              { value: 'rsi', label: t('heatmap:view.rsi') },
            ]}
          />
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
          {view === 'rsi' ? (
            <>
              <Field>
                <span id="rsi-interval-label">{t('heatmap:rsi.interval')}</span>
                <Select
                  aria-labelledby="rsi-interval-label"
                  value={rsiInterval}
                  size="middle"
                  style={{ minWidth: 88 }}
                  options={RSI_HEAT_INTERVALS.map((iv) => ({ value: iv, label: iv }))}
                  onChange={setRsiInterval}
                />
              </Field>
              <Field>
                <span id="rsi-top-label">{t('heatmap:rsi.top')}</span>
                <Select
                  aria-labelledby="rsi-top-label"
                  value={rsiTop}
                  size="middle"
                  style={{ minWidth: 88 }}
                  options={RSI_HEAT_TOPS.map((n) => ({ value: n, label: String(n) }))}
                  onChange={(v) => setRsiTop(Number(v))}
                />
              </Field>
            </>
          ) : (
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
          )}
          <Button
            type="default"
            icon={fullscreen ? <CompressOutlined /> : <ExpandOutlined />}
            onClick={toggleFullscreen}
          >
            {fullscreen ? t('heatmap:exitFullscreen') : t('heatmap:fullscreen')}
          </Button>
        </ToolbarRight>
      </Toolbar>

      {view === 'price' && listQuery.isError ? (
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
      {view === 'rsi' && rsiQuery.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('heatmap:rsi.loadFailed')}
          description={rtkErrorMessage(rsiQuery.error, { resource: t('heatmap:rsi.resource') })}
          action={
            <Button size="small" onClick={() => void rsiQuery.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      <BoardWrap ref={boardRef}>
        {view === 'rsi' ? (
          <RSIHeatmap
            data={rsiData}
            isLoading={rsiQuery.isLoading || rsiQuery.isFetching}
            onOpen={(ex, sym) => navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(sym)}`)}
          />
        ) : (
          <PriceChangeHeatmap
            items={items}
            metric={metric}
            isLoading={listQuery.isLoading || listQuery.isFetching}
            onOpen={(ex, sym) => navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(sym)}`)}
          />
        )}
      </BoardWrap>
    </PageStack>
  );
}
