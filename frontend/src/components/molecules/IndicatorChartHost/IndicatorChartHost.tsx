import { useEffect, useMemo, useRef } from 'react';
import {
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type LineData,
  type Time,
} from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { palette, semanticColors } from '@/styles/tokens';
import {
  BAND_LINE_COLOR,
  DEFAULT_BANDS,
  DEFAULT_HEIGHT,
  RSI_LINE_COLOR,
} from './IndicatorChartHost.constants';
import { ChartContainer } from './IndicatorChartHost.styles';
import type { IndicatorChartHostProps } from './IndicatorChartHost.types';

function toRsiLineData(data: IndicatorChartHostProps['data']): LineData<Time>[] {
  if (!data.length) return [];
  const sorted = [...data]
    .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.value))
    .sort((a, b) => a.time - b.time);
  const out: LineData<Time>[] = [];
  let lastTime: number | null = null;
  for (const p of sorted) {
    if (lastTime !== null && p.time === lastTime) {
      out[out.length - 1] = { time: p.time as Time, value: p.value };
      continue;
    }
    out.push({ time: p.time as Time, value: p.value });
    lastTime = p.time;
  }
  return out;
}

/**
 * RSI (0–100) line chart with optional band guides (default 30 / 70).
 */
export function IndicatorChartHost({
  data,
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
  bands = DEFAULT_BANDS,
}: IndicatorChartHostProps) {
  const { t } = useTranslation('common');
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const hasFittedRef = useRef(false);
  // Content-based key so parent `bands={[30,70]}` each render does not recreate.
  const bandsKey = useMemo(() => bands.join(','), [bands]);
  const dataRef = useRef(data);
  dataRef.current = data;

  useEffect(() => {
    if (isLoading) return;
    const el = containerRef.current;
    if (!el) return;

    const bandLevels = bandsKey
      ? bandsKey.split(',').map(Number).filter(Number.isFinite)
      : [...DEFAULT_BANDS];

    const chart = createChart(el, {
      height,
      layout: {
        background: { color: palette.richBlack },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: semanticColors.chart.grid },
        horzLines: { color: semanticColors.chart.grid },
      },
      width: el.clientWidth,
      rightPriceScale: {
        borderColor: semanticColors.border.default,
        scaleMargins: { top: 0.1, bottom: 0.1 },
      },
      timeScale: {
        borderColor: semanticColors.border.default,
      },
    });

    const series = chart.addSeries(LineSeries, {
      color: RSI_LINE_COLOR,
      lineWidth: 2,
      priceLineVisible: false,
      lastValueVisible: true,
    });

    // Fixed 0–100 scale for RSI readability
    series.applyOptions({
      autoscaleInfoProvider: () => ({
        priceRange: { minValue: 0, maxValue: 100 },
      }),
    });

    for (const level of bandLevels) {
      series.createPriceLine({
        price: level,
        color: BAND_LINE_COLOR,
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: String(level),
      });
    }

    // Apply current data immediately so recreate (bands) never leaves an empty chart.
    const initial = toRsiLineData(dataRef.current);
    series.setData(initial);
    hasFittedRef.current = false;
    if (initial.length) {
      chart.timeScale().fitContent();
      hasFittedRef.current = true;
    }

    chartRef.current = chart;
    seriesRef.current = series;

    const ro = new ResizeObserver(() => {
      if (containerRef.current && chartRef.current) {
        chartRef.current.applyOptions({ width: containerRef.current.clientWidth });
      }
    });
    ro.observe(el);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      hasFittedRef.current = false;
    };
    // height applied separately; bands via content key
    // eslint-disable-next-line react-hooks/exhaustive-deps -- height has its own effect
  }, [isLoading, bandsKey]);

  useEffect(() => {
    chartRef.current?.applyOptions({ height });
  }, [height]);

  useEffect(() => {
    if (isLoading || !seriesRef.current) return;
    const lineData = toRsiLineData(data);
    seriesRef.current.setData(lineData);
    // Fit only once so live poll ticks do not reset user pan/zoom.
    if (lineData.length > 0 && !hasFittedRef.current) {
      chartRef.current?.timeScale().fitContent();
      hasFittedRef.current = true;
    }
  }, [data, isLoading]);

  if (isLoading) {
    return (
      <Skeleton
        variant="chart"
        height={height}
        className={className}
        aria-label={t('a11y.loadingIndicatorChart')}
      />
    );
  }

  return (
    <ChartContainer
      className={className}
      ref={containerRef}
      $height={height}
      data-testid="indicator-chart-host"
    />
  );
}
