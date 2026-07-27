import { useEffect, useLayoutEffect, useRef } from 'react';
import { View, Platform } from 'react-native';
import { useTranslation } from 'react-i18next';
import {
  CandlestickSeries,
  LineSeries,
  LineStyle,
  createChart,
  createSeriesMarkers,
  type IChartApi,
  type IPriceLine,
  type ISeriesApi,
  type ISeriesMarkersPluginApi,
  type LogicalRange,
  type SeriesMarker,
  type Time,
  type UTCTimestamp,
} from 'lightweight-charts';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { colors, semanticColors } from '@/styles/tokens';
import type { CandleChartProps } from './CandleChart.types';
import { styles } from './CandleChart.styles';

const DEFAULT_HISTORY_EDGE = 20;

const LINE_STYLE_MAP: Record<0 | 1 | 2 | 3 | 4, LineStyle> = {
  0: LineStyle.Solid,
  1: LineStyle.Dotted,
  2: LineStyle.Dashed,
  3: LineStyle.LargeDashed,
  4: LineStyle.SparseDotted,
};

/**
 * OHLCV chart host using TradingView Lightweight Charts (v5).
 * Panning left near the oldest bar requests older history from the parent.
 * Optional pump markers + margin price lines.
 * Primary target: react-native-web (Chrome).
 */
export function CandleChart({
  candles,
  overlays = [],
  markers = [],
  priceLines = [],
  height = 260,
  isLoading,
  isLoadingOlder = false,
  errorMessage,
  emptyMessage,
  seriesKey,
  onRequestOlderHistory,
  canLoadOlder = true,
  historyEdgeBars = DEFAULT_HISTORY_EDGE,
}: CandleChartProps) {
  const { t } = useTranslation('detail');
  const resolvedEmpty = emptyMessage ?? t('noCandleData');
  const hostRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const overlayRefs = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null);
  const priceLineObjsRef = useRef<IPriceLine[]>([]);
  const hasFittedRef = useRef(false);
  const prevFirstTimeRef = useRef<number | null>(null);
  const prevLenRef = useRef(0);
  const onRequestOlderRef = useRef(onRequestOlderHistory);
  const canLoadOlderRef = useRef(canLoadOlder);
  const isLoadingOlderRef = useRef(isLoadingOlder);
  const historyEdgeRef = useRef(historyEdgeBars);
  /** Avoid recursive history loads from setData / fitContent range events. */
  const suppressHistoryRequestRef = useRef(false);
  /** Candle count available to range checks (avoids stale chart after setData). */
  const candleCountRef = useRef(0);

  onRequestOlderRef.current = onRequestOlderHistory;
  canLoadOlderRef.current = canLoadOlder;
  isLoadingOlderRef.current = isLoadingOlder;
  historyEdgeRef.current = historyEdgeBars;
  candleCountRef.current = candles.length;

  const showHost =
    Platform.OS === 'web' && candles.length > 0 && !(errorMessage && candles.length === 0);

  const clampEmptyLeftIfExhausted = () => {
    const chart = chartRef.current;
    if (!chart) return;
    if (canLoadOlderRef.current || isLoadingOlderRef.current) return;
    const range = chart.timeScale().getVisibleLogicalRange();
    if (!range || range.from >= 0) return;
    const width = Math.max(range.to - range.from, 10);
    const last = Math.max(candleCountRef.current - 1, 0);
    const from = 0;
    const to = Math.min(last, Math.max(width, 10));
    suppressHistoryRequestRef.current = true;
    try {
      chart.timeScale().setVisibleLogicalRange({ from, to });
    } catch {
      /* ignore invalid range */
    } finally {
      requestAnimationFrame(() => {
        suppressHistoryRequestRef.current = false;
      });
    }
  };

  const maybeRequestOlder = () => {
    if (suppressHistoryRequestRef.current) return;
    if (!canLoadOlderRef.current || isLoadingOlderRef.current) {
      clampEmptyLeftIfExhausted();
      return;
    }
    const chart = chartRef.current;
    const series = candleRef.current;
    if (!chart || !series) return;
    const range = chart.timeScale().getVisibleLogicalRange();
    if (!range) return;

    // Overscrolled into empty left — always fetch more history.
    if (range.from < 0) {
      onRequestOlderRef.current?.();
      return;
    }

    const info = series.barsInLogicalRange(range);
    // barsBefore: how many bars exist left of the visible window.
    // null when range has no intersection — treat as need more.
    if (info == null || info.barsBefore < historyEdgeRef.current) {
      onRequestOlderRef.current?.();
    }
  };

  // Create / recreate chart once the host div is in the DOM.
  useLayoutEffect(() => {
    if (!showHost) return;
    const el = hostRef.current;
    if (!el) return;

    hasFittedRef.current = false;
    prevFirstTimeRef.current = null;
    prevLenRef.current = 0;
    overlayRefs.current.clear();
    markersPluginRef.current = null;
    priceLineObjsRef.current = [];

    const chart = createChart(el, {
      height,
      width: Math.max(el.clientWidth || 0, 320),
      layout: {
        background: { color: semanticColors.bg.muted },
        textColor: semanticColors.text.secondary,
      },
      grid: {
        vertLines: { color: 'rgba(112, 125, 125, 0.2)' },
        horzLines: { color: 'rgba(112, 125, 125, 0.2)' },
      },
      rightPriceScale: { borderVisible: false },
      timeScale: {
        borderVisible: false,
        rightOffset: 4,
        // Allow denser bars so higher TFs still fill the width after fitContent.
        minBarSpacing: 2,
        fixLeftEdge: false,
        fixRightEdge: false,
      },
      handleScroll: {
        mouseWheel: true,
        pressedMouseMove: true,
        horzTouchDrag: true,
        vertTouchDrag: false,
      },
      handleScale: {
        axisPressedMouseMove: true,
        mouseWheel: true,
        pinch: true,
      },
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
    markersPluginRef.current = createSeriesMarkers(candleSeries, []);

    const onRange = (_range: LogicalRange | null) => {
      maybeRequestOlder();
    };
    chart.timeScale().subscribeVisibleLogicalRangeChange(onRange);

    const ro =
      typeof ResizeObserver !== 'undefined'
        ? new ResizeObserver(() => {
            if (hostRef.current && chartRef.current) {
              const w = hostRef.current.clientWidth;
              if (w > 0) chartRef.current.applyOptions({ width: w });
            }
          })
        : null;
    if (ro) ro.observe(el);

    return () => {
      chart.timeScale().unsubscribeVisibleLogicalRangeChange(onRange);
      ro?.disconnect();
      try {
        markersPluginRef.current?.detach();
      } catch {
        /* chart already removed */
      }
      markersPluginRef.current = null;
      priceLineObjsRef.current = [];
      chart.remove();
      chartRef.current = null;
      candleRef.current = null;
      overlayRefs.current.clear();
    };
    // maybeRequestOlder is stable via refs
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [height, seriesKey, showHost]);

  // Push candle + overlay data into an existing chart instance.
  useEffect(() => {
    if (!showHost) return;
    if (!candleRef.current || !chartRef.current) return;
    if (candles.length === 0) return;

    const chart = chartRef.current;
    const firstTime = candles[0]?.time ?? null;
    const prevFirst = prevFirstTimeRef.current;
    const prevLen = prevLenRef.current;
    const prepended =
      firstTime != null &&
      prevFirst != null &&
      firstTime < prevFirst &&
      candles.length >= prevLen;

    // Prefer time-range anchor so overscroll empty space fills with new bars.
    const timeBefore = chart.timeScale().getVisibleRange();
    const logicalBefore = chart.timeScale().getVisibleLogicalRange();

    suppressHistoryRequestRef.current = true;
    try {
      candleRef.current.setData(
        candles.map((c) => ({
          time: c.time as UTCTimestamp,
          open: c.open,
          high: c.high,
          low: c.low,
          close: c.close,
        })),
      );

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

      // Pump markers (must use bar times present in the series).
      const candleTimes = new Set(candles.map((c) => c.time));
      const lwcMarkers: SeriesMarker<UTCTimestamp>[] = markers
        .filter((m) => candleTimes.has(m.time))
        .map((m) => ({
          time: m.time as UTCTimestamp,
          position: m.position,
          shape: m.shape,
          color: m.color,
          text: m.text,
          id: m.id,
          size: m.size,
        }));
      markersPluginRef.current?.setMarkers(lwcMarkers);

      // Margin / level price lines — replace fully when props change.
      const series = candleRef.current;
      if (series) {
        for (const pl of priceLineObjsRef.current) {
          try {
            series.removePriceLine(pl);
          } catch {
            /* already gone */
          }
        }
        priceLineObjsRef.current = priceLines.map((line) =>
          series.createPriceLine({
            price: line.price,
            color: line.color,
            title: line.title,
            lineWidth: (line.lineWidth ?? 1) as 1 | 2 | 3 | 4,
            lineStyle: LINE_STYLE_MAP[line.lineStyle ?? 2],
            axisLabelVisible: line.axisLabelVisible ?? true,
          }),
        );
      }

      if (!hasFittedRef.current) {
        chart.timeScale().fitContent();
        hasFittedRef.current = true;
      } else if (prepended && timeBefore) {
        // Keep the same wall-clock window so newly loaded bars fill empty left.
        try {
          chart.timeScale().setVisibleRange(timeBefore);
        } catch {
          // Fall through to logical handling below.
          if (logicalBefore && Number.isFinite(logicalBefore.from)) {
            const added = Math.max(0, candles.length - prevLen);
            let from = logicalBefore.from + added;
            let to = logicalBefore.to + added;
            if (from < 0) {
              const width = Math.max(to - from, 10);
              from = 0;
              to = Math.min(candles.length - 1, width);
            }
            chart.timeScale().setVisibleLogicalRange({ from, to });
          }
        }
        // If time range still starts before first bar (not enough history),
        // clamp logical left so the chart is full of available bars.
        const afterLogical = chart.timeScale().getVisibleLogicalRange();
        if (afterLogical && afterLogical.from < 0) {
          const width = Math.max(afterLogical.to - afterLogical.from, 10);
          const to = Math.min(candles.length - 1, Math.max(width, 10));
          chart.timeScale().setVisibleLogicalRange({ from: 0, to });
        }
      } else if (logicalBefore && Number.isFinite(logicalBefore.from)) {
        // Poll/update or non-prepend growth: restore logical viewport so the
        // newest bar can enter the right edge without a jump.
        try {
          chart.timeScale().setVisibleLogicalRange(logicalBefore);
        } catch {
          /* ignore */
        }
      } else if (timeBefore) {
        try {
          chart.timeScale().setVisibleRange(timeBefore);
        } catch {
          /* ignore */
        }
      }
    } finally {
      // Double rAF: LWC applies setData layout asynchronously; re-check edge after.
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          suppressHistoryRequestRef.current = false;
          maybeRequestOlder();
        });
      });
    }

    prevFirstTimeRef.current = firstTime;
    prevLenRef.current = candles.length;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [candles, overlays, markers, priceLines, showHost]);

  // When loading finishes and we still need history, try again (chain pages).
  useEffect(() => {
    if (!isLoadingOlder && canLoadOlder && showHost) {
      const id = requestAnimationFrame(() => {
        requestAnimationFrame(() => maybeRequestOlder());
      });
      return () => cancelAnimationFrame(id);
    }
    if (!isLoadingOlder && !canLoadOlder && showHost) {
      const id = requestAnimationFrame(() => clampEmptyLeftIfExhausted());
      return () => cancelAnimationFrame(id);
    }
    return undefined;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoadingOlder, canLoadOlder, showHost, candles.length]);

  if (isLoading && candles.length === 0) {
    return (
      <View style={styles.card}>
        <View style={styles.center}>
          <Skeleton height={height - 40} width="100%" />
        </View>
      </View>
    );
  }

  if (errorMessage && candles.length === 0) {
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
            {resolvedEmpty}
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
            {t('chartWebOnly')}
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.card}>
      <div ref={hostRef} style={{ width: '100%', height }} />
      {isLoadingOlder ? (
        <Text variant="caption" color="steel" style={styles.olderHint}>
          {t('loadingOlderHistory')}
        </Text>
      ) : null}
    </View>
  );
}
