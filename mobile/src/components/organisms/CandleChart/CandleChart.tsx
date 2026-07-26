import { useEffect, useRef } from 'react';
import { View, Platform } from 'react-native';
import {
  CandlestickSeries,
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
} from 'lightweight-charts';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { colors, semanticColors } from '@/styles/tokens';
import type { CandleChartProps } from './CandleChart.types';
import { styles } from './CandleChart.styles';

/**
 * OHLCV chart host using TradingView Lightweight Charts (v5).
 * Primary target: react-native-web (Chrome). Attaches to a DOM div.
 */
export function CandleChart({
  candles,
  overlays = [],
  height = 260,
  isLoading,
  errorMessage,
  emptyMessage = 'No candle data',
}: CandleChartProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());

  useEffect(() => {
    if (Platform.OS !== 'web' || isLoading) return;
    const el = hostRef.current;
    if (!el) return;

    const chart = createChart(el, {
      height,
      width: el.clientWidth || 320,
      layout: {
        background: { color: semanticColors.bg.muted },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: 'rgba(112, 125, 125, 0.2)' },
        horzLines: { color: 'rgba(112, 125, 125, 0.2)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: { borderVisible: false },
    });

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: colors.mountainMeadow ?? '#4FD4A5',
      downColor: semanticColors.status.error,
      borderVisible: false,
      wickUpColor: colors.mountainMeadow ?? '#4FD4A5',
      wickDownColor: semanticColors.status.error,
    });

    chartRef.current = chart;
    candleRef.current = candleSeries;

    const ro =
      typeof ResizeObserver !== 'undefined'
        ? new ResizeObserver(() => {
            if (hostRef.current && chartRef.current) {
              chartRef.current.applyOptions({ width: hostRef.current.clientWidth });
            }
          })
        : null;
    if (ro) ro.observe(el);

    return () => {
      ro?.disconnect();
      chart.remove();
      chartRef.current = null;
      candleRef.current = null;
      overlayRefs.current.clear();
    };
  }, [height, isLoading]);

  useEffect(() => {
    if (isLoading || !candleRef.current || !chartRef.current) return;
    candleRef.current.setData(
      candles.map((c) => ({
        time: c.time as UTCTimestamp,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      })),
    );

    const chart = chartRef.current;
    const seen = new Set(overlays.map((o) => o.id));
    for (const [id, series] of overlayRefs.current) {
      if (!seen.has(id)) {
        chart.removeSeries(series);
        overlayRefs.current.delete(id);
      }
    }
    for (const overlay of overlays) {
      let series = overlayRefs.current.get(overlay.id);
      if (!series) {
        series = chart.addSeries(LineSeries, {
          color: overlay.color,
          lineWidth: 2,
          title: overlay.title,
        });
        overlayRefs.current.set(overlay.id, series);
      } else {
        series.applyOptions({ color: overlay.color, title: overlay.title });
      }
      series.setData(
        overlay.data.map((p) => ({
          time: p.time as UTCTimestamp,
          value: p.value,
        })),
      );
    }
    chart.timeScale().fitContent();
  }, [candles, overlays, isLoading]);

  if (isLoading && candles.length === 0) {
    return (
      <View style={styles.card}>
        <View style={styles.center}>
          <Skeleton height={height - 40} width="100%" />
        </View>
      </View>
    );
  }

  if (errorMessage) {
    return (
      <View style={styles.card}>
        <View style={styles.center}>
          <Text variant="body" color="error">
            {errorMessage}
          </Text>
        </View>
      </View>
    );
  }

  if (!candles.length) {
    return (
      <View style={styles.card}>
        <View style={styles.center}>
          <Text variant="body" color="secondary">
            {emptyMessage}
          </Text>
        </View>
      </View>
    );
  }

  if (Platform.OS !== 'web') {
    return (
      <View style={styles.card}>
        <View style={styles.center}>
          <Text variant="body" color="secondary">
            Chart requires web runtime
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.card}>
      <div ref={hostRef} style={{ width: '100%', height }} />
    </View>
  );
}
