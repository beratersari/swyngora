import { useEffect, useRef } from 'react';
import {
  CandlestickSeries,
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type CandlestickData,
  type LineData,
  type Time,
} from 'lightweight-charts';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { CandleChartHostProps } from './CandleChartHost.types';
import { DEFAULT_HEIGHT } from './CandleChartHost.constants';
import { ChartContainer } from './CandleChartHost.styles';
import { palette, semanticColors } from '@/styles/tokens';

/**
 * Thin host for TradingView Lightweight Charts candlesticks.
 * Optional line overlays (EMA) share the price scale.
 * Supports `isLoading` → chart skeleton.
 */
export function CandleChartHost({
  data,
  overlays = [],
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
}: CandleChartHostProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());

  useEffect(() => {
    if (isLoading) {
      return;
    }

    const el = containerRef.current;
    if (!el) {
      return;
    }

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
      },
      timeScale: {
        borderColor: semanticColors.border.default,
      },
    });

    const series = chart.addSeries(CandlestickSeries, {
      upColor: semanticColors.chart.up,
      downColor: semanticColors.chart.down,
      borderVisible: false,
      wickUpColor: semanticColors.chart.up,
      wickDownColor: semanticColors.chart.down,
    });

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
      overlayRefs.current.clear();
    };
  }, [height, isLoading]);

  useEffect(() => {
    if (isLoading || !seriesRef.current || !chartRef.current) {
      return;
    }
    const bars: CandlestickData<Time>[] = data.map((d) => ({
      time: d.time as Time,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
    }));
    seriesRef.current.setData(bars);

    const chart = chartRef.current;
    const nextIds = new Set(overlays.map((o) => o.id));

    for (const [id, series] of overlayRefs.current) {
      if (!nextIds.has(id)) {
        chart.removeSeries(series);
        overlayRefs.current.delete(id);
      }
    }

    for (const overlay of overlays) {
      let line = overlayRefs.current.get(overlay.id);
      if (!line) {
        line = chart.addSeries(LineSeries, {
          color: overlay.color,
          lineWidth: 2,
          title: overlay.title ?? overlay.id,
          priceLineVisible: false,
          lastValueVisible: true,
        });
        overlayRefs.current.set(overlay.id, line);
      } else {
        line.applyOptions({
          color: overlay.color,
          title: overlay.title ?? overlay.id,
        });
      }
      const lineData: LineData<Time>[] = overlay.data.map((p) => ({
        time: p.time as Time,
        value: p.value,
      }));
      line.setData(lineData);
    }

    chart.timeScale().fitContent();
  }, [data, overlays, isLoading]);

  if (isLoading) {
    return (
      <Skeleton variant="chart" height={height} className={className} aria-label="Loading chart" />
    );
  }

  return (
    <ChartContainer
      className={className}
      ref={containerRef}
      $height={height}
      data-testid="candle-chart-host"
    />
  );
}
