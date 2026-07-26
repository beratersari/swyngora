import { useEffect, useRef } from 'react';
import {
  CandlestickSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type CandlestickData,
  type Time,
} from 'lightweight-charts';
import { Skeleton } from '@/components/atoms/Skeleton';
import type { CandleChartHostProps } from './CandleChartHost.types';
import { DEFAULT_HEIGHT } from './CandleChartHost.constants';
import { ChartContainer } from './CandleChartHost.styles';
import { colors, semanticColors } from '@/styles/tokens';

/**
 * Thin host for TradingView Lightweight Charts.
 * Supports `isLoading` → chart skeleton.
 */
export function CandleChartHost({
  data,
  height = DEFAULT_HEIGHT,
  className,
  isLoading = false,
}: CandleChartHostProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);

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
        background: { color: colors.navy },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: semanticColors.chart.grid },
        horzLines: { color: semanticColors.chart.grid },
      },
      width: el.clientWidth,
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
    };
  }, [height, isLoading]);

  useEffect(() => {
    if (isLoading || !seriesRef.current) {
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
    chartRef.current?.timeScale().fitContent();
  }, [data, isLoading]);

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
