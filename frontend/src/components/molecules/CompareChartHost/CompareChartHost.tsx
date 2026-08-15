import { useEffect, useRef } from 'react';
import {
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type Time,
} from 'lightweight-charts';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { semanticColors } from '@/styles/tokens';
import { DEFAULT_HEIGHT } from './CompareChartHost.constants';
import {
  ChartContainer,
  ChartShell,
  ChartSkeletonLayer,
} from './CompareChartHost.styles';
import type { CompareChartHostProps } from './CompareChartHost.types';

/**
 * Multi-line % change chart for compare mode (2–3 pairs).
 */
export function CompareChartHost({
  series,
  height = DEFAULT_HEIGHT,
  isLoading = false,
  className,
}: CompareChartHostProps) {
  const { t } = useTranslation('common');
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const chart = createChart(el, {
      height,
      autoSize: true,
      layout: {
        background: { color: semanticColors.chart.background },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: semanticColors.chart.grid },
        horzLines: { color: semanticColors.chart.grid },
      },
      rightPriceScale: { borderColor: semanticColors.border.default },
      timeScale: { borderColor: semanticColors.border.default },
    });
    chartRef.current = chart;
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
      seriesRefs.current.clear();
    };
  }, [height]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    const nextIds = new Set(series.map((s) => s.id));
    for (const [id, line] of seriesRefs.current) {
      if (!nextIds.has(id)) {
        chart.removeSeries(line);
        seriesRefs.current.delete(id);
      }
    }
    for (const s of series) {
      let line = seriesRefs.current.get(s.id);
      if (!line) {
        line = chart.addSeries(LineSeries, {
          color: s.color,
          lineWidth: 2,
          title: s.title,
          priceLineVisible: false,
        });
        seriesRefs.current.set(s.id, line);
      } else {
        line.applyOptions({ color: s.color, title: s.title });
      }
      const data = s.data
        .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.value))
        .map((p) => ({ time: p.time as Time, value: p.value }));
      line.setData(data);
    }
    if (series.some((s) => s.data.length > 0)) {
      chart.timeScale().fitContent();
    }
  }, [series]);

  return (
    <ChartShell className={className} $height={height}>
      {isLoading ? (
        <ChartSkeletonLayer>
          <Skeleton variant="chart" height={height} aria-label={t('a11y.loadingChart')} />
        </ChartSkeletonLayer>
      ) : null}
      <ChartContainer ref={containerRef} $height={height} $hidden={isLoading} />
    </ChartShell>
  );
}
