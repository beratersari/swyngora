import { useMemo, useState } from 'react';
import { Button, Select } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import {
  CompareChartHost,
  SERIES_COLORS,
  type CompareSeries,
} from '@/components/molecules/CompareChartHost';
import {
  rtkErrorMessage,
  useGetCandlesQuery,
  type MarketExchange,
} from '@/libs/api';
import {
  MAX_COMPARE_PAIRS,
  apiCandlesToChart,
  closesToPercentSeries,
  comparePairKey,
  filterValidApiCandles,
  formatSymbolDisplay,
  parseComparePairsParam,
  rtkCurrent,
  rtkCurrentPending,
  serializeComparePairs,
  type ComparePair,
} from '@/libs/utils';
import { useDocumentVisible, useMediaQuery } from '@/libs/hooks';
import { mediaQueries } from '@/styles/tokens';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_INTERVAL,
  DEFAULT_DETAIL_SERIES_POLL_MS,
  DESK_COMPARE_CHART_HEIGHT,
  PHONE_COMPARE_CHART_HEIGHT,
} from '@/config/constants';
import {
  EmptyHint,
  ErrorHint,
  Field,
  Legend,
  LegendItem,
  PageStack,
  Toolbar,
} from './ComparePage.styles';

function usePairPercentSeries(
  pair: ComparePair | undefined,
  interval: string,
  color: string,
): { series: CompareSeries | null; isLoading: boolean; isError: boolean; error: unknown } {
  const visible = useDocumentVisible();
  const q = useGetCandlesQuery(
    {
      exchange: (pair?.exchange ?? 'binance') as MarketExchange,
      symbol: pair?.symbol ?? '',
      interval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
    },
    {
      skip: !pair?.symbol,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const series = useMemo(() => {
    const candlesRaw = rtkCurrent(q)?.candles;
    if (!pair || !candlesRaw) return null;
    const candles = filterValidApiCandles(candlesRaw);
    const chart = apiCandlesToChart(candles);
    const pct = closesToPercentSeries(
      chart.map((c) => ({ time: c.time, close: c.close })),
    );
    return {
      id: comparePairKey(pair),
      title: `${formatSymbolDisplay(pair.symbol)} (${pair.exchange})`,
      color,
      data: pct,
    } satisfies CompareSeries;
  }, [pair, q.currentData, color]);
  return {
    series,
    isLoading: rtkCurrentPending(q),
    isError: q.isError,
    error: q.error,
  };
}

export function ComparePage() {
  const { t } = useTranslation(['compare', 'common']);
  const isPhone = useMediaQuery(mediaQueries.phone);
  const [searchParams, setSearchParams] = useSearchParams();
  const pairs = useMemo(
    () => parseComparePairsParam(searchParams.get('pairs')),
    [searchParams],
  );
  const interval = searchParams.get('interval') || DEFAULT_DETAIL_INTERVAL;

  const [draftExchange, setDraftExchange] = useState<MarketExchange>('binance');
  const [draftSymbol, setDraftSymbol] = useState('');

  const p0 = usePairPercentSeries(pairs[0], interval, SERIES_COLORS[0]!);
  const p1 = usePairPercentSeries(pairs[1], interval, SERIES_COLORS[1]!);
  const p2 = usePairPercentSeries(pairs[2], interval, SERIES_COLORS[2]!);

  const seriesList = useMemo(
    () => [p0.series, p1.series, p2.series].filter(Boolean) as CompareSeries[],
    [p0.series, p1.series, p2.series],
  );

  const loading = p0.isLoading || p1.isLoading || p2.isLoading;
  const firstError = p0.isError ? p0.error : p1.isError ? p1.error : p2.isError ? p2.error : null;

  const setPairs = (next: ComparePair[]) => {
    const sp = new URLSearchParams(searchParams);
    const ser = serializeComparePairs(next);
    if (ser) sp.set('pairs', ser);
    else sp.delete('pairs');
    if (interval && interval !== DEFAULT_DETAIL_INTERVAL) sp.set('interval', interval);
    else if (sp.get('interval') === DEFAULT_DETAIL_INTERVAL) sp.delete('interval');
    setSearchParams(sp, { replace: true });
  };

  const addPair = () => {
    const symbol = draftSymbol.trim().toUpperCase();
    if (!symbol || pairs.length >= MAX_COMPARE_PAIRS) return;
    const next = [...pairs, { exchange: draftExchange, symbol }];
    const seen = new Set<string>();
    const deduped: ComparePair[] = [];
    for (const p of next) {
      const k = comparePairKey(p);
      if (seen.has(k)) continue;
      seen.add(k);
      deduped.push(p);
    }
    setPairs(deduped.slice(0, MAX_COMPARE_PAIRS));
    setDraftSymbol('');
  };

  return (
    <PageStack>
      <PageHeader title={t('compare:title')} />

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary">
            {t('compare:exchange')}
          </Text>
          <Select
            value={draftExchange}
            style={{ minWidth: 120 }}
            options={['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'].map((e) => ({ value: e, label: e }))}
            onChange={(v) => setDraftExchange(v)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('compare:symbol')}
          </Text>
          <SymbolSuggest
            exchange={draftExchange}
            value={draftSymbol}
            onChange={setDraftSymbol}
            aria-label={t('compare:symbol')}
            placeholder="BTCUSDT"
            style={{ minWidth: 160 }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('compare:interval')}
          </Text>
          <Select
            value={interval}
            style={{ minWidth: 100 }}
            options={['15m', '1h', '4h', '1d'].map((iv) => ({ value: iv, label: iv }))}
            onChange={(iv) => {
              const sp = new URLSearchParams(searchParams);
              sp.set('interval', iv);
              setSearchParams(sp, { replace: true });
            }}
          />
        </Field>
        <Button
          type="primary"
          disabled={!draftSymbol.trim() || pairs.length >= MAX_COMPARE_PAIRS}
          onClick={addPair}
        >
          {t('compare:add')}
        </Button>
      </Toolbar>

      <Legend>
        {seriesList.map((s) => (
          <LegendItem key={s.id} $color={s.color}>
            <Text variant="caption" color="secondary">
              {s.title}
            </Text>
            <Button
              size="small"
              type="link"
              onClick={() => setPairs(pairs.filter((p) => comparePairKey(p) !== s.id))}
            >
              {t('compare:remove')}
            </Button>
          </LegendItem>
        ))}
      </Legend>

      {pairs.length === 0 ? (
        <EmptyHint>
          <Text variant="body" color="secondary">
            {t('compare:emptyHint')}
          </Text>
        </EmptyHint>
      ) : null}

      {firstError ? (
        <ErrorHint>
          <Text variant="label" color="primary">
            {t('compare:loadFailed')}
          </Text>
          <Text variant="caption" color="secondary">
            {rtkErrorMessage(firstError, { resource: t('compare:resource') })}
          </Text>
        </ErrorHint>
      ) : null}

      <CompareChartHost
        series={seriesList}
        isLoading={loading && seriesList.length === 0}
        height={isPhone ? PHONE_COMPARE_CHART_HEIGHT : DESK_COMPARE_CHART_HEIGHT}
      />
      <Text variant="caption" color="secondary">
        {t('compare:axisHint')}
      </Text>
    </PageStack>
  );
}
