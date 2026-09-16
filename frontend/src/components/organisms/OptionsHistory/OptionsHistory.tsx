import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Segmented } from 'antd';
import { createChart, LineSeries, type IChartApi, type ISeriesApi, type Time } from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { semanticColors } from '@/styles/tokens';
import { OPTIONS_HISTORY_METRICS, OPTIONS_HISTORY_RANGES } from './constants';
import { historyMetricHasData, historyWindow, toHistoryLine, windowChipLabel } from './helpers';
import { ChartBox, Chip, Chips, Stack, Toolbar } from './OptionsHistory.styles';
import type { OptionsHistoryMetric, OptionsHistoryProps, OptionsHistoryRange } from './OptionsHistory.types';

export function OptionsHistory({ data, isLoading }: OptionsHistoryProps) {
  const { t } = useTranslation('options');
  const [range, setRange] = useState<OptionsHistoryRange>('48h');
  const [metric, setMetric] = useState<OptionsHistoryMetric>('oi');
  const boxRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const compareRef = useRef<ISeriesApi<'Line'> | null>(null);
  const lineRef = useRef<ReturnType<typeof toHistoryLine>>([]);
  const compareLineRef = useRef<ReturnType<typeof toHistoryLine>>([]);

  const history = data?.history;
  const line = useMemo(
    () => toHistoryLine(history?.points, metric, range),
    [history?.points, metric, range],
  );
  const compareLine = useMemo(
    () => (metric === 'flip' ? toHistoryLine(history?.points, 'index', range) : []),
    [history?.points, metric, range],
  );
  lineRef.current = line;
  compareLineRef.current = compareLine;
  const ready = line.length > 0;
  const win24 = historyWindow(history?.windows, '24h');

  useLayoutEffect(() => {
    if (!ready) return;
    const el = boxRef.current;
    if (!el) return;
    const chart = createChart(el, {
      width: Math.max(el.clientWidth || 320, 8),
      height: Math.max(el.clientHeight || 200, 8),
      layout: {
        background: { color: 'transparent' },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: 'rgba(79, 212, 165, 0.08)' },
        horzLines: { color: 'rgba(79, 212, 165, 0.08)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: {
        borderVisible: false,
        timeVisible: range !== '7d',
        secondsVisible: false,
      },
      handleScroll: false,
      handleScale: false,
    });
    const series = chart.addSeries(LineSeries, {
      color: semanticColors.chart.emaFast,
      lineWidth: 2,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    const compare = chart.addSeries(LineSeries, {
      color: semanticColors.chart.neutral,
      lineWidth: 2,
      lastValueVisible: true,
      priceLineVisible: false,
    });
    chartRef.current = chart;
    seriesRef.current = series;
    compareRef.current = compare;
    const initial = lineRef.current;
    if (initial.length > 0) {
      series.setData(initial.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    const overlay = compareLineRef.current;
    if (overlay.length > 0) {
      compare.setData(overlay.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    if (initial.length > 0 || overlay.length > 0) {
      chart.timeScale().fitContent();
    }
    const ro = new ResizeObserver(() => {
      const host = boxRef.current;
      const live = chartRef.current;
      if (!host || !live) return;
      const width = host.clientWidth;
      const height = host.clientHeight;
      if (width < 8 || height < 8) return;
      live.applyOptions({ width, height });
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      compareRef.current = null;
    };
  }, [ready, range, metric]);

  useEffect(() => {
    const series = seriesRef.current;
    const compare = compareRef.current;
    const chart = chartRef.current;
    if (!series || !chart || line.length === 0) return;
    series.setData(line.map((p) => ({ time: p.time as Time, value: p.value })));
    if (compare) {
      compare.setData(compareLine.map((p) => ({ time: p.time as Time, value: p.value })));
    }
    chart.timeScale().fitContent();
  }, [line, compareLine]);

  if (isLoading && !data) {
    return <Skeleton height={240} />;
  }
  if (!history || (history.points?.length ?? 0) === 0) {
    return (
      <Text variant="bodySm" color="secondary">
        {t('history.empty')}
      </Text>
    );
  }

  const metricReady = historyMetricHasData(history.points, metric);

  return (
    <Stack>
      <Chips>
        {(['1h', '4h', '24h', '7d'] as const).map((id) => {
          const win = historyWindow(history.windows, id);
          const chip = windowChipLabel(win, metric);
          let label = chip.delta;
          if (chip.note === 'regime' && chip.to) {
            label = t(`history.regimeTo.${chip.to}` as 'history.regimeTo.short');
          } else if (chip.note === 'flipSide' && chip.to) {
            label = t(`history.spot.${chip.to}` as 'history.spot.above');
          }
          return (
            <Chip key={id}>
              <Text variant="caption" color="secondary">
                {t(`history.window.${id}`)}
              </Text>
              <Text variant="bodySm" color="primary">
                {label || t('history.collecting')}
              </Text>
            </Chip>
          );
        })}
      </Chips>

      <Toolbar>
        <Segmented
          size="small"
          value={metric}
          options={OPTIONS_HISTORY_METRICS.map((value) => ({
            value,
            label: t(`history.metric.${value}`),
          }))}
          onChange={(next) => setMetric(String(next) as OptionsHistoryMetric)}
          aria-label={t('history.metricAria')}
        />
        <Segmented
          size="small"
          value={range}
          options={OPTIONS_HISTORY_RANGES.map((value) => ({
            value,
            label: t(`history.range.${value}`),
          }))}
          onChange={(next) => setRange(String(next) as OptionsHistoryRange)}
          aria-label={t('history.rangeAria')}
        />
      </Toolbar>

      {!metricReady ? (
        <Text variant="bodySm" color="secondary">
          {metric === 'iv'
            ? t('history.emptyIv')
            : metric === 'gex' || metric === 'flip'
              ? t('history.emptyGamma')
              : t('history.collectingSeries')}
        </Text>
      ) : (
        <ChartBox ref={boxRef} data-testid="options-history-chart" />
      )}

      {win24?.oi?.changePct || win24?.dvol?.changePct ? (
        <Text variant="caption" color="secondary">
          {t('history.hint')}
        </Text>
      ) : null}
    </Stack>
  );
}
